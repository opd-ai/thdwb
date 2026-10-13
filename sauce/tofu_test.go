package sauce

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTOFUStorePersistsAndMatchesExactCertificate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tofu.json")
	store, err := NewTOFUStore(path)
	require.NoError(t, err)
	cert := tofuTestCertificate(t, "example.com", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, store.TrustCertificate(" EXAMPLE.COM. ", cert))
	require.True(t, store.IsTrusted("example.com", cert))

	otherCert := tofuTestCertificate(t, "example.com", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.False(t, store.IsTrusted("example.com", otherCert))
	require.Error(t, store.TrustCertificate("example.com", otherCert))
	require.True(t, store.IsTrusted("example.com", cert))
	require.False(t, store.IsTrusted("other.example", cert))

	reloaded, err := NewTOFUStore(path)
	require.NoError(t, err)
	require.True(t, reloaded.IsTrusted("example.com.", cert))
	pins := reloaded.List()
	require.Len(t, pins, 1)
	require.Equal(t, "example.com", pins[0].Host)
	require.Equal(t, hex.EncodeToString(sha256Sum(cert.Raw)), pins[0].Fingerprint)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestTOFUStoreNormalizesAndValidatesHosts(t *testing.T) {
	store, err := NewTOFUStore("")
	require.NoError(t, err)
	cert := tofuTestCertificate(t, "example.com", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, store.TrustCertificate("  ExAmPlE.CoM. ", cert))

	for _, host := range []string{"", "example.com:443", "https://example.com", "-bad.example", "bad..example", "*.example.com", "fe80::1%eth0", "192.0.2.1%eth0"} {
		t.Run(host, func(t *testing.T) {
			require.Error(t, store.TrustCertificate(host, cert))
			require.False(t, store.IsTrusted(host, cert))
		})
	}
	require.NoError(t, store.TrustCertificate("192.0.2.1", tofuTestCertificate(t, "192.0.2.1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))))
}

func TestTOFUStoreCanonicalizesIPHostsAndChecksIPSubjectAltName(t *testing.T) {
	store, err := NewTOFUStore("")
	require.NoError(t, err)
	cert := tofuTestCertificate(t, "2001:0db8:0:0::1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, store.TrustCertificate("2001:0DB8::1", cert))
	require.True(t, store.IsTrusted("2001:db8:0:0::1", cert))
	require.Equal(t, "2001:db8::1", store.List()[0].Host)
	require.Error(t, store.TrustCertificate("2001:db8::2", cert))
	require.False(t, store.IsTrusted("2001:db8::2", cert))
}

func TestTOFUStoreRejectsInvalidCertificates(t *testing.T) {
	store, err := NewTOFUStore("")
	require.NoError(t, err)
	now := time.Now()
	valid := tofuTestCertificate(t, "example.com", now.Add(-time.Hour), now.Add(time.Hour))
	expired := tofuTestCertificate(t, "example.com", now.Add(-2*time.Hour), now.Add(-time.Hour))
	notYetValid := tofuTestCertificate(t, "example.com", now.Add(time.Hour), now.Add(2*time.Hour))
	wrongHost := tofuTestCertificate(t, "other.example", now.Add(-time.Hour), now.Add(time.Hour))

	for _, cert := range []*x509.Certificate{nil, expired, notYetValid, wrongHost} {
		require.Error(t, store.TrustCertificate("example.com", cert))
	}
	nonSelfSigned := tofuTestNonSelfSignedCertificate(t, valid)
	require.Error(t, store.TrustCertificate("example.com", nonSelfSigned))
	require.Empty(t, store.List())
}

func TestTOFUStoreRemove(t *testing.T) {
	store, err := NewTOFUStore("")
	require.NoError(t, err)
	cert := tofuTestCertificate(t, "example.com", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.ErrorIs(t, store.RemoveCertificate("example.com"), ErrTOFUHostNotFound)
	require.NoError(t, store.TrustCertificate("example.com", cert))
	require.NoError(t, store.RemoveCertificate("EXAMPLE.COM."))
	require.Empty(t, store.List())
	require.ErrorIs(t, store.RemoveCertificate("example.com"), ErrTOFUHostNotFound)
}

func TestTOFUStoreConcurrentAccess(t *testing.T) {
	store, err := NewTOFUStore("")
	require.NoError(t, err)
	cert := tofuTestCertificate(t, "example.com", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, store.TrustCertificate("example.com", cert))

	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				require.True(t, store.IsTrusted("EXAMPLE.com.", cert))
				require.Len(t, store.List(), 1)
				if i%2 == 0 {
					require.NoError(t, store.TrustCertificate("example.com", cert))
				}
			}
		}(i)
	}
	workers.Wait()
	require.Len(t, store.List(), 1)
}

func tofuTestCertificate(t *testing.T, host string, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: host},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func tofuTestNonSelfSignedCertificate(t *testing.T, valid *x509.Certificate) *x509.Certificate {
	t.Helper()
	issuer, issuerKey := tofuTestIssuer(t)
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano() + 2),
		Subject:      valid.Subject,
		DNSNames:     valid.DNSNames,
		NotBefore:    valid.NotBefore,
		NotAfter:     valid.NotAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, issuer, &leafKey.PublicKey, issuerKey)
	require.NoError(t, err)
	leaf, err := x509.ParseCertificate(leafDER)
	require.NoError(t, err)
	return leaf
}

func tofuTestIssuer(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano() + 1),
		Subject:               pkix.Name{CommonName: "issuer"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, key
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}
