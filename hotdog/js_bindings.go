package hotdog

import (
	"fmt"
	"net/url"

	"github.com/dop251/goja"
)

// JSDOMWrapper wraps a NodeDOM for JavaScript access
type JSDOMWrapper struct {
	node      *NodeDOM
	runtime   *goja.Runtime
	windowCtx *WindowContext
}

// JSDocumentWrapper wraps a Document for JavaScript access
type JSDocumentWrapper struct {
	doc       *Document
	runtime   *goja.Runtime
	windowCtx *WindowContext
}

// JSWindowWrapper wraps a WindowContext for JavaScript access
type JSWindowWrapper struct {
	windowCtx *WindowContext
	runtime   *goja.Runtime
}

// JSConsoleWrapper provides console.log, console.error, console.warn
type JSConsoleWrapper struct {
	windowCtx *WindowContext
	runtime   *goja.Runtime
}

// InitJSRuntime initializes the JavaScript runtime with DOM bindings
func (wc *WindowContext) InitJSRuntime() error {
	runtime := wc.GetJSRuntime()
	if runtime == nil {
		return fmt.Errorf("no JavaScript runtime available")
	}

	// Create console object
	console := &JSConsoleWrapper{
		windowCtx: wc,
		runtime:   runtime,
	}
	consoleObj := runtime.NewObject()
	consoleObj.Set("log", console.log)
	consoleObj.Set("error", console.error)
	consoleObj.Set("warn", console.warn)
	runtime.Set("console", consoleObj)

	// Create document object
	if wc.ActiveDocument != nil {
		docWrapper := &JSDocumentWrapper{
			doc:       wc.ActiveDocument,
			runtime:   runtime,
			windowCtx: wc,
		}
		docObj := runtime.NewObject()
		docObj.Set("querySelector", docWrapper.querySelector)
		docObj.Set("querySelectorAll", docWrapper.querySelectorAll)
		docObj.Set("getElementById", docWrapper.getElementById)
		docObj.Set("getElementsByClassName", docWrapper.getElementsByClassName)
		docObj.Set("getElementsByTagName", docWrapper.getElementsByTagName)
		docObj.Set("createElement", docWrapper.createElement)
		docObj.Set("createTextNode", docWrapper.createTextNode)
		docObj.Set("body", docWrapper.getBody)
		docObj.Set("documentElement", docWrapper.getDocumentElement)
		docObj.Set("title", docWrapper.getTitle)
		runtime.Set("document", docObj)
	}

	// Create window object
	windowWrapper := &JSWindowWrapper{
		windowCtx: wc,
		runtime:   runtime,
	}
	windowObj := runtime.NewObject()
	windowObj.Set("document", runtime.Get("document"))
	windowObj.Set("console", runtime.Get("console"))
	windowObj.Set("location", windowWrapper.getLocation)
	windowObj.Set("localStorage", windowWrapper.getLocalStorage)
	windowObj.Set("sessionStorage", windowWrapper.getSessionStorage)
	windowObj.Set("addEventListener", windowWrapper.addEventListener)
	windowObj.Set("removeEventListener", windowWrapper.removeEventListener)
	windowObj.Set("postMessage", windowWrapper.postMessage)
	windowObj.Set("fetch", windowWrapper.fetch)
	windowObj.Set("XMLHttpRequest", windowWrapper.XMLHttpRequest)
	runtime.Set("window", windowObj)
	runtime.Set("self", windowObj) // window.self === window

	// Block dangerous APIs unless CSP allows
	runtime.Set("eval", func(call goja.FunctionCall) goja.Value {
		// Check CSP for 'unsafe-eval' - for now, always block
		panic(runtime.NewGoError(SecurityError("eval() is blocked by CSP")))
	})
	runtime.Set("Function", func(call goja.FunctionCall) goja.Value {
		// Check CSP for 'unsafe-eval' - for now, always block
		panic(runtime.NewGoError(SecurityError("Function constructor is blocked by CSP")))
	})

	return nil
}

// === JSConsoleWrapper methods ===

func (c *JSConsoleWrapper) log(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	Log(fmt.Sprintf("[console.log] [origin: %s]", c.windowCtx.GetOrigin().String()), fmt.Sprint(args...))
	return goja.Undefined()
}

func (c *JSConsoleWrapper) error(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	Log(fmt.Sprintf("[console.error] [origin: %s]", c.windowCtx.GetOrigin().String()), fmt.Sprint(args...))
	return goja.Undefined()
}

func (c *JSConsoleWrapper) warn(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	Log(fmt.Sprintf("[console.warn] [origin: %s]", c.windowCtx.GetOrigin().String()), fmt.Sprint(args...))
	return goja.Undefined()
}
