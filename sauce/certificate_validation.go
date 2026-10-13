package sauce

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var defaultTOFUStoreState struct {
	once  sync.Once
	store *TOFUStore
	err   error
}

// DefaultTOFUStore returns the persistent certificate store used by default requests.
func DefaultTOFUStore() (*TOFUStore, error) {
	defaultTOFUStoreState.once.Do(func() {
		var configDir string
		configDir, defaultTOFUStoreState.err = os.UserConfigDir()
		if defaultTOFUStoreState.err != nil {
			return
		}
		path := filepath.Join(configDir, "thdwb", "tofu-certificates.json")
		defaultTOFUStoreState.store, defaultTOFUStoreState.err = NewTOFUStore(path)
	})
	return defaultTOFUStoreState.store, defaultTOFUStoreState.err
}

func newVerifiedTransport(base http.RoundTripper, store *TOFUStore, hostname string) (http.RoundTripper, error) {
	transport, ok := base.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("HTTPS requires a configurable *http.Transport, got %T", base)
	}
	config, err := verifiedTLSConfig(transport.TLSClientConfig, store, hostname)
	if err != nil {
		return nil, err
	}
	clone := transport.Clone()
	clone.TLSClientConfig = config
	return clone, nil
}

func verifiedTLSConfig(original *tls.Config, store *TOFUStore, hostname string) (*tls.Config, error) {
	config := &tls.Config{}
	if original != nil {
		config = original.Clone()
	}
	if config.InsecureSkipVerify {
		return nil, errors.New("HTTPS transport has InsecureSkipVerify enabled")
	}
	config.VerifyConnection = certificateVerifier(config, store, hostname)
	return config, nil
}

func certificateVerifier(config *tls.Config, store *TOFUStore, hostname string) func(tls.ConnectionState) error {
	originalPeerVerify := config.VerifyPeerCertificate
	originalConnectionVerify := config.VerifyConnection
	config.InsecureSkipVerify = true
	config.VerifyPeerCertificate = nil
	return func(state tls.ConnectionState) error {
		chains, err := verifyCertificateChain(state.PeerCertificates, hostname, store)
		if err != nil {
			return err
		}
		return runCustomTLSVerifiers(state, chains, originalPeerVerify, originalConnectionVerify)
	}
}

func runCustomTLSVerifiers(state tls.ConnectionState, chains [][]*x509.Certificate, peerVerify func([][]byte, [][]*x509.Certificate) error, connectionVerify func(tls.ConnectionState) error) error {
	if peerVerify != nil {
		if err := peerVerify(rawCertificates(state.PeerCertificates), chains); err != nil {
			return fmt.Errorf("custom peer certificate validation failed: %w", err)
		}
	}
	if connectionVerify != nil {
		state.VerifiedChains = chains
		if err := connectionVerify(state); err != nil {
			return fmt.Errorf("custom TLS connection validation failed: %w", err)
		}
	}
	return nil
}

func verifyCertificateChain(certificates []*x509.Certificate, hostname string, store *TOFUStore) ([][]*x509.Certificate, error) {
	if len(certificates) == 0 {
		return nil, errors.New("TLS peer provided no certificates")
	}
	if hostname == "" {
		return nil, errors.New("HTTPS certificate validation requires a hostname")
	}
	hostname, err := normalizeTOFUHost(hostname)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPS certificate hostname: %w", err)
	}
	options := x509.VerifyOptions{DNSName: hostname, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	intermediates := x509.NewCertPool()
	for _, certificate := range certificates[1:] {
		intermediates.AddCert(certificate)
	}
	options.Intermediates = intermediates
	chains, err := certificates[0].Verify(options)
	if err == nil {
		return chains, nil
	}
	if !isSelfSignedCertificate(certificates[0]) || store == nil || !store.IsTrusted(hostname, certificates[0]) {
		return nil, fmt.Errorf("TLS certificate validation failed for %q: %w", hostname, err)
	}
	options.Roots = x509.NewCertPool()
	options.Roots.AddCert(certificates[0])
	chains, err = certificates[0].Verify(options)
	if err != nil {
		return nil, fmt.Errorf("pinned TLS certificate validation failed for %q: %w", hostname, err)
	}
	return chains, nil
}

func isSelfSignedCertificate(certificate *x509.Certificate) bool {
	return certificate != nil &&
		bytes.Equal(certificate.RawIssuer, certificate.RawSubject) &&
		certificate.CheckSignature(certificate.SignatureAlgorithm, certificate.RawTBSCertificate, certificate.Signature) == nil
}

func rawCertificates(certificates []*x509.Certificate) [][]byte {
	raw := make([][]byte, len(certificates))
	for i, certificate := range certificates {
		raw[i] = certificate.Raw
	}
	return raw
}
