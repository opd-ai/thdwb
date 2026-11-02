// Package multiprocess implements isolated extension processes separate from
// the renderer, with IPC communication between the browser shell and each
// extension process.
package multiprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ProcessState describes the execution state of an extension process.
type ProcessState int

const (
	// ProcessStopped means the extension process is not running.
	ProcessStopped ProcessState = iota
	// ProcessStarting means the process is being launched.
	ProcessStarting
	// ProcessRunning means the process is running and accepting IPC messages.
	ProcessRunning
	// ProcessStopping means the process is being shut down.
	ProcessStopping
	// ProcessCrashed means the process exited unexpectedly.
	ProcessCrashed
)

// ExtensionProcess represents an isolated OS process running a single
// extension's background script / service worker. Each extension runs in its
// own process with its own Goja VM, isolated from the renderer and other
// extensions.
type ExtensionProcess struct {
	mu       sync.Mutex
	extID    string
	cmd      *exec.Cmd
	state    ProcessState
	ipc      *IPCConnection
	exitChan chan struct{}
	stopChan chan struct{}

	// startTime records when the process was last started.
	startTime time.Time
	// restartCount tracks how many times the process has been restarted.
	restartCount int
	// maxRestarts is the maximum number of restart attempts before giving up.
	maxRestarts int
}

// NewExtensionProcess creates a new extension process for the given extension
// ID. The process is not started until Start is called.
func NewExtensionProcess(extID string) *ExtensionProcess {
	return &ExtensionProcess{
		extID:      extID,
		state:      ProcessStopped,
		exitChan:   make(chan struct{}),
		stopChan:   make(chan struct{}),
		maxRestarts: 3,
	}
}

// ID returns the extension ID this process runs.
func (p *ExtensionProcess) ID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.extID
}

// State returns the current process state.
func (p *ExtensionProcess) State() ProcessState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// IsRunning reports whether the process is currently running.
func (p *ExtensionProcess) IsRunning() bool {
	return p.State() == ProcessRunning
}

// Start launches the extension process. It blocks until the process is
// running or fails to start.
func (p *ExtensionProcess) Start() error {
	p.mu.Lock()
	if p.state == ProcessRunning || p.state == ProcessStarting {
		p.mu.Unlock()
		return nil // already running or starting
	}
	p.state = ProcessStarting
	p.startTime = time.Now()
	p.mu.Unlock()

	// Build the extension process command.
	exe, err := os.Executable()
	if err != nil {
		p.mu.Lock()
		p.state = ProcessStopped
		p.mu.Unlock()
		return fmt.Errorf("get executable path: %w", err)
	}

	args := []string{
		"-ext-process",
		"-ext-id=" + p.extID,
	}

	p.cmd = exec.Command(exe, args...)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr

	// Set up IPC connection.
	ipc, err := NewIPCConnection(p.cmd)
	if err != nil {
		p.mu.Lock()
		p.state = ProcessStopped
		p.mu.Unlock()
		return fmt.Errorf("create IPC connection: %w", err)
	}
	p.ipc = ipc

	// Start the process.
	if err := p.cmd.Start(); err != nil {
		p.mu.Lock()
		p.state = ProcessStopped
		p.mu.Unlock()
		return fmt.Errorf("start extension process %s: %w", p.extID, err)
	}

	// Wait for the process to signal readiness via IPC.
	ready, err := p.ipc.WaitForReady(10 * time.Second)
	if err != nil || !ready {
		p.Stop()
		return fmt.Errorf("extension process %s did not become ready: %w", p.extID, err)
	}

	p.mu.Lock()
	p.state = ProcessRunning
	p.mu.Unlock()

	// Monitor process exit in a goroutine.
	go p.monitorExit()

	return nil
}

// monitorExit watches the extension process and transitions to crashed state
// if it exits unexpectedly.
func (p *ExtensionProcess) monitorExit() {
	done := make(chan error, 1)
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		<-waitDone // ensure the wait goroutine has finished
		p.mu.Lock()
		if p.state == ProcessStopping || p.state == ProcessStopped {
			p.mu.Unlock()
			return // expected shutdown
		}
		p.state = ProcessCrashed
		close(p.exitChan)
		p.mu.Unlock()

		// Attempt restart if under the limit.
		if err != nil && p.restartCount < p.maxRestarts {
			p.restartCount++
			time.Sleep(time.Second * time.Duration(p.restartCount))
			_ = p.Start()
		}
	case <-p.stopChan:
		// Process was stopped intentionally. The wait goroutine will
		// complete on its own once the process exits.
		<-waitDone
	}
}

// Stop shuts down the extension process gracefully. It waits for the process
// to exit or kills it after a timeout.
func (p *ExtensionProcess) Stop() error {
	p.mu.Lock()
	if p.state == ProcessStopped || p.state == ProcessStopping {
		p.mu.Unlock()
		return nil
	}
	p.state = ProcessStopping
	close(p.stopChan)
	p.mu.Unlock()

	if p.ipc != nil {
		_ = p.ipc.Close()
	}

	if p.cmd != nil && p.cmd.Process != nil {
		// Give the process a short grace period to exit.
		done := make(chan struct{})
		go func() {
			_ = p.cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Process exited gracefully.
		case <-time.After(5 * time.Second):
			// Force kill.
			_ = p.cmd.Process.Kill()
		}
	}

	p.mu.Lock()
	p.state = ProcessStopped
	p.cmd = nil
	p.ipc = nil
	p.mu.Unlock()

	return nil
}

// Restart stops and restarts the extension process.
func (p *ExtensionProcess) Restart() error {
	if err := p.Stop(); err != nil {
		return err
	}
	p.mu.Lock()
	p.restartCount = 0
	p.mu.Unlock()
	return p.Start()
}

// IPC returns the IPC connection for this process, or nil if not connected.
func (p *ExtensionProcess) IPC() *IPCConnection {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ipc
}

// StartTime returns when the process was last started.
func (p *ExtensionProcess) StartTime() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.startTime
}

// RestartCount returns the number of times the process has been restarted.
func (p *ExtensionProcess) RestartCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.restartCount
}

// SetMaxRestarts sets the maximum number of restart attempts.
func (p *ExtensionProcess) SetMaxRestarts(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRestarts = n
}

// IPCMessage is a message sent between the browser shell and an extension
// process over the IPC channel.
type IPCMessage struct {
	// Type identifies the message kind.
	Type string `json:"type"`
	// ExtID is the extension ID the message is for.
	ExtID string `json:"ext_id,omitempty"`
	// Payload is the message body, JSON-encoded.
	Payload json.RawMessage `json:"payload,omitempty"`
	// Seq is a sequence number for request/response matching.
	Seq uint64 `json:"seq,omitempty"`
}

// IPCConnection manages the bidirectional IPC channel with an extension
// process via stdin/stdout.
type IPCConnection struct {
	mu      sync.Mutex
	conn    *json.Encoder
	decoder *json.Decoder
	seq     uint64
	pending map[uint64]chan IPCMessage
	ready   chan struct{}
	closed  bool
}

// NewIPCConnection creates an IPC connection wired to the given command's
// stdin/stdout pipes. The caller must call cmd.Start() before using the
// connection.
func NewIPCConnection(cmd *exec.Cmd) (*IPCConnection, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	ipc := &IPCConnection{
		conn:    json.NewEncoder(stdin),
		decoder: json.NewDecoder(stdout),
		pending: make(map[uint64]chan IPCMessage),
		ready:   make(chan struct{}),
	}

	// Start the read loop.
	go ipc.readLoop()

	return ipc, nil
}

// readLoop reads IPC messages from the extension process and dispatches
// them to waiting callers or the ready channel.
func (ipc *IPCConnection) readLoop() {
	for {
		var msg IPCMessage
		if err := ipc.decoder.Decode(&msg); err != nil {
			// Connection closed.
			ipc.mu.Lock()
			ipc.closed = true
			for _, ch := range ipc.pending {
				close(ch)
			}
			ipc.pending = nil
			ipc.mu.Unlock()
			return
		}

		ipc.mu.Lock()
		if msg.Type == "ready" {
			close(ipc.ready)
			ipc.mu.Unlock()
			continue
		}
		if ch, ok := ipc.pending[msg.Seq]; ok {
			delete(ipc.pending, msg.Seq)
			ipc.mu.Unlock()
			ch <- msg
			continue
		}
		ipc.mu.Unlock()
		// Unsolicited messages from the extension process are ignored for now.
	}
}

// Send sends an IPC message to the extension process and waits for a
// response with the matching sequence number.
func (ipc *IPCConnection) Send(msgType string, payload interface{}) (IPCMessage, error) {
	ipc.mu.Lock()
	if ipc.closed {
		ipc.mu.Unlock()
		return IPCMessage{}, fmt.Errorf("IPC connection closed")
	}
	ipc.seq++
	seq := ipc.seq

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		ipc.mu.Unlock()
		return IPCMessage{}, fmt.Errorf("marshal payload: %w", err)
	}

	msg := IPCMessage{
		Type:    msgType,
		Payload: payloadBytes,
		Seq:     seq,
	}

	ch := make(chan IPCMessage, 1)
	ipc.pending[seq] = ch
	ipc.mu.Unlock()

	if err := ipc.conn.Encode(msg); err != nil {
		ipc.mu.Lock()
		delete(ipc.pending, seq)
		close(ch)
		ipc.mu.Unlock()
		return IPCMessage{}, fmt.Errorf("send IPC message: %w", err)
	}

	// Wait for response with a timeout.
	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(30 * time.Second):
		ipc.mu.Lock()
		delete(ipc.pending, seq)
		close(ch)
		ipc.mu.Unlock()
		return IPCMessage{}, fmt.Errorf("IPC request timed out")
	}
}

// Close closes the IPC connection.
func (ipc *IPCConnection) Close() error {
	ipc.mu.Lock()
	if ipc.closed {
		ipc.mu.Unlock()
		return nil
	}
	ipc.closed = true
	for _, ch := range ipc.pending {
		close(ch)
	}
	ipc.pending = nil
	ipc.mu.Unlock()
	return nil
}

// WaitForReady waits for the extension process to signal readiness.
func (ipc *IPCConnection) WaitForReady(timeout time.Duration) (bool, error) {
	select {
	case <-ipc.ready:
		return true, nil
	case <-time.After(timeout):
		return false, fmt.Errorf("timeout waiting for ready")
	}
}

// EnsureExtProcessFlag ensures the binary re-executes as an extension process
// when launched with -ext-process. This should be called early in main() to
// handle the extension process entry point.
func EnsureExtProcessFlag(extID string, runExtProcess func(extID string) error) bool {
	for _, arg := range os.Args[1:] {
		if arg == "-ext-process" {
			// Parse the -ext-id flag.
			id := extID
			for _, a := range os.Args[1:] {
				if len(a) > 8 && a[:8] == "-ext-id=" {
					id = a[8:]
					break
				}
			}
			if err := runExtProcess(id); err != nil {
				fmt.Fprintf(os.Stderr, "extension process %s failed: %v\n", id, err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	return false
}

// ProcessManager manages multiple extension processes, starting and stopping
// them on demand.
type ProcessManager struct {
	mu      sync.Mutex
	processes map[string]*ExtensionProcess
}

// NewProcessManager creates a new process manager.
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[string]*ExtensionProcess),
	}
}

// GetOrCreate returns the extension process for the given ID, creating it if
// needed.
func (pm *ProcessManager) GetOrCreate(extID string) *ExtensionProcess {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if p, ok := pm.processes[extID]; ok {
		return p
	}
	p := NewExtensionProcess(extID)
	pm.processes[extID] = p
	return p
}

// Start starts the extension process for the given extension ID.
func (pm *ProcessManager) Start(extID string) error {
	p := pm.GetOrCreate(extID)
	return p.Start()
}

// Stop stops the extension process for the given extension ID.
func (pm *ProcessManager) Stop(extID string) error {
	pm.mu.Lock()
	p, ok := pm.processes[extID]
	if ok {
		delete(pm.processes, extID)
	}
	pm.mu.Unlock()
	if !ok {
		return nil
	}
	return p.Stop()
}

// StopAll stops all running extension processes.
func (pm *ProcessManager) StopAll() error {
	pm.mu.Lock()
	procs := make([]*ExtensionProcess, 0, len(pm.processes))
	for _, p := range pm.processes {
		procs = append(procs, p)
	}
	pm.processes = make(map[string]*ExtensionProcess)
	pm.mu.Unlock()

	var firstErr error
	for _, p := range procs {
		if err := p.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// IsRunning reports whether the extension process for the given ID is running.
func (pm *ProcessManager) IsRunning(extID string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	p, ok := pm.processes[extID]
	if !ok {
		return false
	}
	return p.IsRunning()
}

// Count returns the number of tracked extension processes.
func (pm *ProcessManager) Count() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.processes)
}

// RunningCount returns the number of running extension processes.
func (pm *ProcessManager) RunningCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	count := 0
	for _, p := range pm.processes {
		if p.IsRunning() {
			count++
		}
	}
	return count
}

// EnsureFilePath returns the absolute path to the current executable, used
// for spawning extension processes.
func EnsureFilePath() (string, error) {
	return os.Executable()
}

// ResolveExtProcessScript resolves the path to a script to run in the
// extension process, relative to the extension directory.
func ResolveExtProcessScript(extDir, scriptPath string) string {
	if filepath.IsAbs(scriptPath) {
		return scriptPath
	}
	return filepath.Join(extDir, scriptPath)
}
