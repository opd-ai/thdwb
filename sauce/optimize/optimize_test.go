package optimize

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSCacheStoreAndLookup(t *testing.T) {
	dc := NewDNSCache(time.Minute)

	ips := []net.IP{net.ParseIP("93.184.216.34")}
	dc.Store("example.com", ips)

	got := dc.Lookup("example.com")
	require.NotNil(t, got)
	assert.Equal(t, ips, got)

	// Missing host returns nil.
	assert.Nil(t, dc.Lookup("missing.com"))
}

func TestDNSCacheExpiry(t *testing.T) {
	dc := NewDNSCache(10 * time.Millisecond)

	ips := []net.IP{net.ParseIP("93.184.216.34")}
	dc.Store("example.com", ips)

	assert.NotNil(t, dc.Lookup("example.com"))

	time.Sleep(20 * time.Millisecond)
	assert.Nil(t, dc.Lookup("example.com"))
}

func TestDNSCacheDeleteAndClear(t *testing.T) {
	dc := NewDNSCache(time.Minute)

	dc.Store("a.com", []net.IP{net.ParseIP("1.1.1.1")})
	dc.Store("b.com", []net.IP{net.ParseIP("2.2.2.2")})
	assert.Equal(t, 2, dc.Count())

	dc.Delete("a.com")
	assert.Nil(t, dc.Lookup("a.com"))
	assert.NotNil(t, dc.Lookup("b.com"))
	assert.Equal(t, 1, dc.Count())

	dc.Clear()
	assert.Equal(t, 0, dc.Count())
}

func TestTLSConfigCache(t *testing.T) {
	tcc := NewTLSConfigCache()

	cfg1 := tcc.Get("example.com")
	require.NotNil(t, cfg1)
	assert.Equal(t, "example.com", cfg1.ServerName)
	assert.EqualValues(t, tls.VersionTLS12, cfg1.MinVersion)

	cfg2 := tcc.Get("example.com")
	assert.Same(t, cfg1, cfg2) // same config returned

	cfg3 := tcc.Get("other.com")
	require.NotNil(t, cfg3)
	assert.NotSame(t, cfg1, cfg3)
}

func TestConnectionPool(t *testing.T) {
	dns := NewDNSCache(time.Minute)
	tls := NewTLSConfigCache()
	cp := NewConnectionPool(dns, tls)

	client1 := cp.GetClient("https")
	require.NotNil(t, client1)

	client2 := cp.GetClient("https")
	assert.Same(t, client1, client2) // same client for same scheme

	client3 := cp.GetClient("http")
	require.NotNil(t, client3)
	assert.NotSame(t, client1, client3) // different scheme = different client

	assert.Equal(t, 2, cp.Count())
}

func TestConnectionPoolCloseIdle(t *testing.T) {
	dns := NewDNSCache(time.Minute)
	tls := NewTLSConfigCache()
	cp := NewConnectionPool(dns, tls)

	cp.GetClient("https")
	cp.GetClient("http")

	// Should not panic.
	cp.CloseIdleConnections()
}

func TestNetworkOptimizer(t *testing.T) {
	no := NewNetworkOptimizer()
	require.NotNil(t, no.DNS())
	require.NotNil(t, no.TLS())
	require.NotNil(t, no.Pool())

	client := no.ClientForScheme("https")
	require.NotNil(t, client)

	// Same client returned for same scheme.
	assert.Same(t, client, no.ClientForScheme("https"))

	assert.Equal(t, 1, no.Pool().Count())
}

func TestDialerWithCache(t *testing.T) {
	dns := NewDNSCache(time.Minute)
	dd := NewDialerWithCache(dns, 5*time.Second)

	// Dial a non-existent host - should fail (no DNS resolution).
	_, err := dd.DialContext(context.Background(), "tcp", "nonexistent.invalid:80")
	assert.Error(t, err)
}

func TestDNSCacheThreadSafety(t *testing.T) {
	dc := NewDNSCache(time.Minute)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(i int) {
			dc.Store("host", []net.IP{net.ParseIP("1.1.1.1")})
			dc.Lookup("host")
			dc.Count()
			if i == 9 {
				close(done)
			}
		}(i)
	}
	<-done
	assert.Equal(t, 1, dc.Count())
}
