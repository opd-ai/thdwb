package sauce

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
)

// InspectTOFUCertificate performs a TLS handshake without sending application data.
// Its certificate and fingerprint can be reviewed before explicitly trusting it.
func InspectTOFUCertificate(ctx context.Context, address string) (*x509.Certificate, string, error) {
	hostname, err := inspectionHostname(address)
	if err != nil {
		return nil, "", err
	}
	connection, err := dialForCertificateInspection(ctx, hostname, address)
	if err != nil {
		return nil, "", err
	}
	defer connection.Close()
	return inspectedCertificate(connection, hostname)
}

func inspectionHostname(address string) (string, error) {
	hostname, _, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("inspect TLS certificate address %q: %w", address, err)
	}
	return normalizeTOFUHost(hostname)
}

func dialForCertificateInspection(ctx context.Context, hostname, address string) (net.Conn, error) {
	config := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	connection, err := (&tls.Dialer{Config: config}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("inspect TLS certificate from %q: %w", hostname, err)
	}
	return connection, nil
}

func inspectedCertificate(connection net.Conn, hostname string) (*x509.Certificate, string, error) {
	tlsConnection, ok := connection.(*tls.Conn)
	if !ok {
		return nil, "", errors.New("TLS dialer returned a non-TLS connection")
	}
	state := tlsConnection.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, "", errors.New("TLS peer provided no certificates")
	}
	certificate := state.PeerCertificates[0]
	fingerprint, err := certificateFingerprint(hostname, certificate)
	if err != nil {
		return nil, "", fmt.Errorf("inspect TLS certificate from %q: %w", hostname, err)
	}
	return certificate, fingerprint, nil
}
