// Package telemetry implements opt-in, privacy-respecting telemetry for thdwb.
// It collects basic usage metrics (page loads, extension usage, session length)
// and submits them to a configurable endpoint only when the user has explicitly
// opted in. No personal data, URLs, or page content is ever collected.
package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OptInState reports whether telemetry is enabled.
type OptInState int

const (
	// NotAsked means the user has not yet been asked to opt in.
	NotAsked OptInState = iota
	// Disabled means the user has explicitly opted out.
	Disabled
	// OptedIn means the user has explicitly opted in.
	OptedIn
)

// Config holds telemetry configuration.
type Config struct {
	// Enabled is the user's opt-in state.
	Enabled OptInState

	// Endpoint is the telemetry collection endpoint URL.
	Endpoint string

	// ClientID is a randomly generated, non-personal identifier.
	ClientID string

	// Interval is how often to submit telemetry data.
	Interval time.Duration

	// HTTPClient is the HTTP client used for submissions.
	HTTPClient *http.Client
}

// DefaultConfig returns a default telemetry configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:   NotAsked,
		Endpoint:  "https://telemetry.thdwb.dev/submit",
		Interval:  15 * time.Minute,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Metric is a single telemetry data point.
type Metric struct {
	// Name is the metric name (e.g., "page_load", "extension_start").
	Name string `json:"name"`

	// Value is the metric value.
	Value float64 `json:"value"`

	// Timestamp is when the metric was recorded.
	Timestamp time.Time `json:"timestamp"`

	// Tags are optional key-value pairs for categorization.
	Tags map[string]string `json:"tags,omitempty"`
}

// Payload is a batch of metrics submitted to the endpoint.
type Payload struct {
	// ClientID is the telemetry client identifier.
	ClientID string `json:"client_id"`

	// Timestamp is when the payload was created.
	Timestamp time.Time `json:"timestamp"`

	// Metrics is the batch of metrics.
	Metrics []Metric `json:"metrics"`
}

// Telemetry collects and submits usage metrics.
type Telemetry struct {
	mu      sync.Mutex
	config  *Config
	metrics []Metric
	store   *Store
}

// New creates a new Telemetry instance.
func New(config *Config, storeDir string) (*Telemetry, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var store *Store
	if storeDir != "" {
		var err error
		store, err = NewStore(storeDir)
		if err != nil {
			return nil, fmt.Errorf("create telemetry store: %w", err)
		}
	}

	t := &Telemetry{
		config:  config,
		metrics: make([]Metric, 0),
		store:   store,
	}

	// Load persisted metrics if a store is configured.
	if store != nil {
		if saved, err := store.Load(); err == nil {
			t.metrics = saved
		}
	}

	return t, nil
}

// MustNew creates a new Telemetry instance, panicking on error.
func MustNew(config *Config, storeDir string) *Telemetry {
	t, err := New(config, storeDir)
	if err != nil {
		panic(err)
	}
	return t
}

// IsEnabled reports whether telemetry is currently enabled (opted in).
func (t *Telemetry) IsEnabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.config.Enabled == OptedIn
}

// IsNotAsked reports whether the user has not yet been asked to opt in.
func (t *Telemetry) IsNotAsked() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.config.Enabled == NotAsked
}

// OptIn enables telemetry.
func (t *Telemetry) OptIn() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.config.Enabled = OptedIn
	if t.store != nil {
		_ = t.store.SaveEnabled(true)
	}
}

// OptOut disables telemetry.
func (t *Telemetry) OptOut() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.config.Enabled = Disabled
	if t.store != nil {
		_ = t.store.SaveEnabled(false)
	}
}

// Record adds a metric to the batch.
func (t *Telemetry) Record(name string, value float64, tags map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.config.Enabled != OptedIn {
		return
	}

	t.metrics = append(t.metrics, Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
	})
}

// RecordPageLoad records a page load metric.
func (t *Telemetry) RecordPageLoad(duration time.Duration, tags map[string]string) {
	t.Record("page_load", duration.Seconds(), tags)
}

// RecordExtensionStart records an extension startup metric.
func (t *Telemetry) RecordExtensionStart(extID string, duration time.Duration) {
	t.Record("extension_start", duration.Seconds(), map[string]string{
		"extension_id": extID,
	})
}

// RecordSessionLength records a browser session length metric.
func (t *Telemetry) RecordSessionLength(duration time.Duration) {
	t.Record("session_length", duration.Seconds(), nil)
}

// Submit sends all pending metrics to the telemetry endpoint.
// It returns an error if submission fails; metrics are retained for retry.
func (t *Telemetry) Submit() error {
	t.mu.Lock()
	if t.config.Enabled != OptedIn {
		t.mu.Unlock()
		return nil // not enabled, nothing to submit
	}

	metrics := make([]Metric, len(t.metrics))
	copy(metrics, t.metrics)
	t.mu.Unlock()

	if len(metrics) == 0 {
		return nil
	}

	payload := Payload{
		ClientID:  t.config.ClientID,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", t.config.Endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("submit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("submit failed: status %d", resp.StatusCode)
	}

	// Clear submitted metrics on success.
	t.mu.Lock()
	t.metrics = make([]Metric, 0)
	if t.store != nil {
		_ = t.store.SaveMetrics(t.metrics)
	}
	t.mu.Unlock()

	return nil
}

// PendingCount returns the number of unsent metrics.
func (t *Telemetry) PendingCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.metrics)
}

// Store persists telemetry state across restarts.
type Store struct {
	dir string
}

// NewStore creates a telemetry store in the given directory.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// SaveEnabled persists the enabled state.
func (s *Store) SaveEnabled(enabled bool) error {
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	return os.WriteFile(filepath.Join(s.dir, "enabled"), []byte(state), 0o600)
}

// SaveMetrics persists metrics (called after successful submit with empty slice).
func (s *Store) SaveMetrics(metrics []Metric) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "metrics.json"), data, 0o600)
}

// Load loads persisted metrics.
func (s *Store) Load() ([]Metric, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, "metrics.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var metrics []Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}
	return metrics, nil
}
