package extensions

import (
	"fmt"
	"os"
	"sync"

	goja "github.com/dop251/goja"
)

// ServiceWorkerState describes the lifecycle state of a service worker.
type ServiceWorkerState int

const (
	// ServiceWorkerInstalling means the service worker script is being
	// parsed and the install event is being dispatched.
	ServiceWorkerInstalling ServiceWorkerState = iota
	// ServiceWorkerInstalled means the install event has completed and the
	// worker is waiting to activate.
	ServiceWorkerInstalled
	// ServiceWorkerActivating means the activate event is being dispatched.
	ServiceWorkerActivating
	// ServiceWorkerActivated means the service worker is fully active and
	// can handle fetch events.
	ServiceWorkerActivated
	// ServiceWorkerRedundant means the service worker is no longer needed
	// and has been discarded.
	ServiceWorkerRedundant
)

// ServiceWorker represents a Manifest v3 service worker for an extension.
// Unlike v2 persistent background scripts, service workers are event-driven
// and terminated when idle.
type ServiceWorker struct {
	mu       sync.Mutex
	extID    string
	vm       *goja.Runtime
	state    ServiceWorkerState
	path     string
	running  bool
	stopChan chan struct{}

	// eventListeners holds registered event listeners keyed by event name.
	eventListeners map[string]goja.Callable
}

// NewServiceWorker creates a new service worker for the given extension.
func NewServiceWorker(extID, path string) *ServiceWorker {
	return &ServiceWorker{
		extID:          extID,
		path:           path,
		state:          ServiceWorkerInstalling,
		stopChan:       make(chan struct{}),
		eventListeners: make(map[string]goja.Callable),
	}
}

// State returns the current lifecycle state of the service worker.
func (sw *ServiceWorker) State() ServiceWorkerState {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.state
}

// SetState transitions the service worker to a new lifecycle state.
func (sw *ServiceWorker) SetState(state ServiceWorkerState) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.state = state
}

// IsRunning reports whether the service worker VM is currently active.
func (sw *ServiceWorker) IsRunning() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.running
}

// Start initializes the service worker VM, dispatches the install event,
// then the activate event, and transitions to the activated state.
func (sw *ServiceWorker) Start(vm *goja.Runtime) error {
	sw.mu.Lock()
	if sw.running {
		sw.mu.Unlock()
		return nil // already running
	}
	sw.vm = vm
	sw.running = true
	sw.mu.Unlock()

	// Install the "self" global on the VM before running the script so that
	// service worker code can call self.addEventListener.
	if err := installSelfGlobal(vm, sw); err != nil {
		sw.mu.Lock()
		sw.running = false
		sw.mu.Unlock()
		return fmt.Errorf("install self global: %w", err)
	}

	// Load and execute the service worker script.
	if err := runScriptInVM(vm, sw.path); err != nil {
		sw.mu.Lock()
		sw.running = false
		sw.mu.Unlock()
		return fmt.Errorf("service worker %s: load script: %w", sw.path, err)
	}

	// Dispatch the install event.
	sw.SetState(ServiceWorkerInstalling)
	if err := sw.dispatchEvent("install"); err != nil {
		sw.mu.Lock()
		sw.running = false
		sw.mu.Unlock()
		return fmt.Errorf("service worker %s: install event: %w", sw.path, err)
	}
	sw.SetState(ServiceWorkerInstalled)

	// Dispatch the activate event.
	sw.SetState(ServiceWorkerActivating)
	if err := sw.dispatchEvent("activate"); err != nil {
		sw.mu.Lock()
		sw.running = false
		sw.mu.Unlock()
		return fmt.Errorf("service worker %s: activate event: %w", sw.path, err)
	}
	sw.SetState(ServiceWorkerActivated)

	return nil
}

// Stop terminates the service worker VM and transitions to the redundant
// state.
func (sw *ServiceWorker) Stop() error {
	sw.mu.Lock()
	sw.running = false
	sw.state = ServiceWorkerRedundant
	if sw.stopChan != nil {
		close(sw.stopChan)
	}
	sw.mu.Unlock()

	// Release the VM reference.
	sw.vm = nil
	return nil
}

// dispatchEvent fires a service worker lifecycle event (install, activate,
// fetch) by calling the corresponding global handler registered in the VM.
// Service workers register handlers via self.addEventListener(event, cb).
func (sw *ServiceWorker) dispatchEvent(event string) error {
	sw.mu.Lock()
	handler := sw.eventListeners[event]
	sw.mu.Unlock()

	if handler == nil {
		return nil
	}

	// Create a minimal event object.
	eventObj := sw.vm.NewObject()
	_ = eventObj.Set("type", event)

	_, err := handler(goja.Undefined(), eventObj)
	return err
}

// String returns a string representation of the service worker state.
type FetchEvent struct {
	RequestURL string
	RequestMethod string
	RequestHeaders map[string]string
	Response     chan *FetchResponse
}

// FetchResponse is the response produced by a service worker fetch handler.
type FetchResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

// DispatchFetch dispatches a fetch event to the service worker and waits
// for a response. If the service worker does not handle the fetch, the
// caller should proceed with the default network fetch.
func (sw *ServiceWorker) DispatchFetch(event FetchEvent) (*FetchResponse, error) {
	sw.mu.Lock()
	if sw.state != ServiceWorkerActivated || sw.vm == nil {
		sw.mu.Unlock()
		return nil, nil // service worker not active; use default fetch
	}
	handler := sw.eventListeners["fetch"]
	sw.mu.Unlock()

	if handler == nil {
		return nil, nil
	}

	eventObj := sw.vm.NewObject()
	_ = eventObj.Set("type", "fetch")
	_ = eventObj.Set("request", map[string]interface{}{
		"url":      event.RequestURL,
		"method":   event.RequestMethod,
		"headers":  event.RequestHeaders,
	})
	_ = eventObj.Set("respondWith", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			respObj := call.Arguments[0].ToObject(sw.vm)
			if respObj != nil {
				resp := &FetchResponse{
					StatusCode: 200,
					Headers:    make(map[string]string),
				}
				if v := respObj.Get("status"); v != nil && v != goja.Undefined() {
					resp.StatusCode = int(v.ToInteger())
				}
				if v := respObj.Get("headers"); v != nil && v != goja.Undefined() {
					if h, ok := v.ToObject(sw.vm).Export().(map[string]interface{}); ok {
						for k, val := range h {
							resp.Headers[k] = fmt.Sprintf("%v", val)
						}
					}
				}
				if v := respObj.Get("body"); v != nil && v != goja.Undefined() {
					resp.Body = v.String()
				}
				event.Response <- resp
			}
		}
		return goja.Undefined()
	})

	_, err := handler(goja.Undefined(), eventObj)
	if err != nil {
		return nil, err
	}

	// Wait for the response (non-blocking if service worker doesn't call respondWith).
	select {
	case resp := <-event.Response:
		return resp, nil
	default:
		return nil, nil
	}
}

// EnsureStopChan returns the stop channel for external coordination.
func (sw *ServiceWorker) EnsureStopChan() <-chan struct{} {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.stopChan
}

// Path returns the service worker script path.
func (sw *ServiceWorker) Path() string {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.path
}

// SetPath updates the service worker script path.
func (sw *ServiceWorker) SetPath(path string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.path = path
}

// String returns a string representation of the service worker state.
func (sw *ServiceWorker) String() string {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return fmt.Sprintf("ServiceWorker{extID: %s, path: %s, state: %d, running: %v}",
		sw.extID, sw.path, sw.state, sw.running)
}

// loadServiceWorkerScript reads the service worker script from disk.
func loadServiceWorkerScript(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read service worker script %s: %w", path, err)
	}
	return string(data), nil
}

// installSelfGlobal installs the "self" global object on the VM so service
// worker scripts can call self.addEventListener(event, cb).
func installSelfGlobal(vm *goja.Runtime, sw *ServiceWorker) error {
	selfObj := vm.NewObject()
	_ = selfObj.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		event := call.Argument(0).String()
		cb, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			return goja.Undefined()
		}
		sw.mu.Lock()
		sw.eventListeners[event] = cb
		sw.mu.Unlock()
		return goja.Undefined()
	})
	_ = vm.Set("self", selfObj)
	return nil
}
