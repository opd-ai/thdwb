package sauce

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyCertificateChainRequiresPinnedSelfSignedCertificate(t *testing.T) {
	certificate := makeTestCertificate(t, "example.test", time.Now().Add(time.Hour))
	store, err := NewTOFUStore("")
	require.NoError(t, err)

	_, err = verifyCertificateChain([]*x509.Certificate{certificate}, "example.test", store)
	require.Error(t, err)
	require.NoError(t, store.TrustCertificate("example.test", certificate))
	_, err = verifyCertificateChain([]*x509.Certificate{certificate}, "example.test", store)
	require.NoError(t, err)

	_, err = verifyCertificateChain([]*x509.Certificate{certificate}, "other.test", store)
	require.Error(t, err)
}

func TestVerifyCertificateChainRejectsExpiredPinnedCertificate(t *testing.T) {
	certificate := makeTestCertificate(t, "example.test", time.Now().Add(time.Second))
	store, err := NewTOFUStore("")
	require.NoError(t, err)

	require.NoError(t, store.TrustCertificate("example.test", certificate))
	time.Sleep(time.Until(certificate.NotAfter) + 10*time.Millisecond)
	_, err = verifyCertificateChain([]*x509.Certificate{certificate}, "example.test", store)
	require.Error(t, err)
}

func TestVerifiedTLSConfigRejectsExistingInsecureSkipVerify(t *testing.T) {
	_, err := verifiedTLSConfig(&tls.Config{InsecureSkipVerify: true}, nil, "example.test")
	require.ErrorContains(t, err, "InsecureSkipVerify")
}

func makeTestCertificate(t *testing.T, hostname string, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(hostname); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{hostname}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return certificate
}

func TestIsSelfSignedCertificateChecksSignature(t *testing.T) {
	certificate := makeTestCertificate(t, "example.test", time.Now().Add(time.Hour))
	assert.True(t, isSelfSignedCertificate(certificate))
	tampered := *certificate
	tampered.Signature = append([]byte(nil), certificate.Signature...)
	tampered.Signature[0] ^= 1
	assert.False(t, isSelfSignedCertificate(&tampered))
}
