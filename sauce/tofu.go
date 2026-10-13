package sauce

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/idna"
)

// ErrTOFUHostNotFound is returned when removing a host that is not trusted.
var ErrTOFUHostNotFound = errors.New("TOFU certificate not found")

// TOFUPin is the normalized hostname and SHA-256 fingerprint of a trusted certificate.
type TOFUPin struct {
	Host        string `json:"host"`
	Fingerprint string `json:"fingerprint"`
}

// TOFUStore stores one trusted certificate fingerprint per hostname or IP.
type TOFUStore struct {
	mu   sync.RWMutex
	path string
	pins map[string]string
}

// NewTOFUStore opens a persistent JSON store, creating it if necessary.
// An empty path creates an in-memory store.
func NewTOFUStore(path string) (*TOFUStore, error) {
	store := &TOFUStore{path: path, pins: make(map[string]string)}
	if path == "" {
		return store, nil
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// TrustCertificate validates and records cert as the trusted certificate for host.
func (store *TOFUStore) TrustCertificate(host string, cert *x509.Certificate) error {
	if store == nil {
		return errors.New("nil TOFU store")
	}
	normalizedHost, err := normalizeTOFUHost(host)
	if err != nil {
		return err
	}
	fingerprint, err := certificateFingerprint(normalizedHost, cert)
	if err != nil {
		return err
	}
	return store.trustFingerprint(normalizedHost, fingerprint)
}

func (store *TOFUStore) trustFingerprint(host, fingerprint string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous, existed := store.pins[host]
	if existed && previous != fingerprint {
		return fmt.Errorf("TOFU certificate already pinned for %q; remove it before trusting a replacement", host)
	}
	if existed {
		return nil
	}
	store.pins[host] = fingerprint
	if err := store.persistLocked(); err != nil {
		delete(store.pins, host)
		return err
	}
	return nil
}

// RemoveCertificate removes the pin for host, returning ErrTOFUHostNotFound if absent.
func (store *TOFUStore) RemoveCertificate(host string) error {
	if store == nil {
		return errors.New("nil TOFU store")
	}
	normalizedHost, err := normalizeTOFUHost(host)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous, exists := store.pins[normalizedHost]
	if !exists {
		return ErrTOFUHostNotFound
	}
	delete(store.pins, normalizedHost)
	if err := store.persistLocked(); err != nil {
		store.pins[normalizedHost] = previous
		return err
	}
	return nil
}

// IsTrusted reports whether cert exactly matches the validated pin for host.
func (store *TOFUStore) IsTrusted(host string, cert *x509.Certificate) bool {
	if store == nil {
		return false
	}
	normalizedHost, err := normalizeTOFUHost(host)
	if err != nil {
		return false
	}
	fingerprint, err := certificateFingerprint(normalizedHost, cert)
	if err != nil {
		return false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.pins[normalizedHost] == fingerprint
}

// List returns the trusted pins in normalized hostname order.
func (store *TOFUStore) List() []TOFUPin {
	if store == nil {
		return nil
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	pins := make([]TOFUPin, 0, len(store.pins))
	for host, fingerprint := range store.pins {
		pins = append(pins, TOFUPin{Host: host, Fingerprint: fingerprint})
	}
	sort.Slice(pins, func(i, j int) bool { return pins[i].Host < pins[j].Host })
	return pins
}

func (store *TOFUStore) load() error {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
			return fmt.Errorf("create TOFU store directory: %w", err)
		}
		return store.persistLocked()
	}
	if err != nil {
		return fmt.Errorf("read TOFU store: %w", err)
	}
	if err := os.Chmod(store.path, 0o600); err != nil {
		return fmt.Errorf("secure TOFU store permissions: %w", err)
	}
	return store.decodePins(data)
}

func (store *TOFUStore) decodePins(data []byte) error {
	var pins []TOFUPin
	if err := json.Unmarshal(data, &pins); err != nil {
		return fmt.Errorf("decode TOFU store: %w", err)
	}
	for _, pin := range pins {
		host, err := normalizeTOFUHost(pin.Host)
		if err != nil || host != pin.Host {
			return fmt.Errorf("invalid host in TOFU store: %q", pin.Host)
		}
		if !validFingerprint(pin.Fingerprint) {
			return fmt.Errorf("invalid fingerprint for TOFU host %q", pin.Host)
		}
		if _, exists := store.pins[host]; exists {
			return fmt.Errorf("duplicate TOFU host %q", host)
		}
		store.pins[host] = pin.Fingerprint
	}
	return nil
}

func (store *TOFUStore) persistLocked() error {
	if store.path == "" {
		return nil
	}
	pins := make([]TOFUPin, 0, len(store.pins))
	for host, fingerprint := range store.pins {
		pins = append(pins, TOFUPin{Host: host, Fingerprint: fingerprint})
	}
	sort.Slice(pins, func(i, j int) bool { return pins[i].Host < pins[j].Host })
	data, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return fmt.Errorf("encode TOFU store: %w", err)
	}
	return writeTOFUFile(store.path, data)
}

func writeTOFUFile(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create TOFU store temporary file: %w", err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := writeTOFUTemporary(temporary, data); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace TOFU store: %w", err)
	}
	return nil
}

func writeTOFUTemporary(temporary *os.File, data []byte) error {
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure TOFU store temporary file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write TOFU store: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync TOFU store: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close TOFU store: %w", err)
	}
	return nil
}

func normalizeTOFUHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("TOFU host must be a DNS hostname or IP address")
	}
	if strings.Contains(host, "%") {
		return "", fmt.Errorf("invalid zone-scoped TOFU host %q", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}
	host = strings.TrimSuffix(host, ".")
	asciiHost, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("invalid TOFU hostname %q: %w", host, err)
	}
	asciiHost = strings.ToLower(asciiHost)
	if len(asciiHost) > 253 {
		return "", fmt.Errorf("invalid TOFU DNS hostname %q", host)
	}
	for _, label := range strings.Split(asciiHost, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("invalid TOFU DNS hostname %q", host)
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return "", fmt.Errorf("invalid TOFU DNS hostname %q", host)
			}
		}
	}
	return asciiHost, nil
}

func certificateFingerprint(host string, cert *x509.Certificate) (string, error) {
	if cert == nil {
		return "", errors.New("TOFU certificate must not be nil")
	}
	if !bytes.Equal(cert.RawSubject, cert.RawIssuer) ||
		cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) != nil {
		return "", errors.New("TOFU certificate must be self-signed")
	}
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return "", errors.New("TOFU certificate is not currently valid")
	}
	if err := cert.VerifyHostname(host); err != nil {
		return "", fmt.Errorf("TOFU certificate is not valid for host %q: %w", host, err)
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}

func validFingerprint(fingerprint string) bool {
	if len(fingerprint) != sha256.Size*2 || strings.ToLower(fingerprint) != fingerprint {
		return false
	}
	_, err := hex.DecodeString(fingerprint)
	return err == nil
}
