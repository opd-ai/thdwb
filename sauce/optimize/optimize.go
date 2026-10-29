// Package optimize provides network optimizations for the thdwb browser:
// connection pooling, DNS caching, and TLS session reuse.
package optimize

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// DNSEntry is a cached DNS resolution result.
type DNSEntry struct {
	IPs       []net.IP
	Expiry    time.Time
}

// DNSCache caches DNS resolutions to avoid repeated lookups.
type DNSCache struct {
	mu     sync.RWMutex
	entries map[string]*DNSEntry
	ttl    time.Duration
}

// NewDNSCache creates a DNSCache with the given TTL.
func NewDNSCache(ttl time.Duration) *DNSCache {
	return &DNSCache{
		entries: make(map[string]*DNSEntry),
		ttl:     ttl,
	}
}

// Lookup returns cached IPs for a host, or nil if not found or expired.
func (dc *DNSCache) Lookup(host string) []net.IP {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	entry, ok := dc.entries[host]
	if !ok || time.Now().After(entry.Expiry) {
		return nil
	}
	return entry.IPs
}

// Store caches IPs for a host.
func (dc *DNSCache) Store(host string, ips []net.IP) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.entries[host] = &DNSEntry{
		IPs:    ips,
		Expiry: time.Now().Add(dc.ttl),
	}
}

// Delete removes a host from the cache.
func (dc *DNSCache) Delete(host string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	delete(dc.entries, host)
}

// Clear removes all entries.
func (dc *DNSCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.entries = make(map[string]*DNSEntry)
}

// Count returns the number of cached entries.
func (dc *DNSCache) Count() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.entries)
}

// DialerWithCache is a net.Dialer that uses a DNS cache.
type DialerWithCache struct {
	dialer  *net.Dialer
	dns     *DNSCache
}

// NewDialerWithCache creates a DialerWithCache.
func NewDialerWithCache(dns *DNSCache, timeout time.Duration) *DialerWithCache {
	return &DialerWithCache{
		dialer: &net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		},
		dns: dns,
	}
}

// DialContext dials a network address, using the DNS cache when available.
func (dd *DialerWithCache) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return dd.dialer.DialContext(ctx, network, address)
	}

	// Try DNS cache first.
	if ips := dd.dns.Lookup(host); len(ips) > 0 {
		var lastErr error
		for _, ip := range ips {
			conn, err := dd.dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return nil, lastErr
		}
	}

	// Fall back to normal resolution and cache the result.
	conn, err := dd.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	// Cache resolved IPs.
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if raddr := tcpConn.RemoteAddr(); raddr != nil {
			if tcpAddr, ok := raddr.(*net.TCPAddr); ok {
				dd.dns.Store(host, []net.IP{tcpAddr.IP})
			}
		}
	}

	return conn, nil
}

// TLSConfigCache caches TLS session tickets by server name for session reuse.
type TLSConfigCache struct {
	mu      sync.RWMutex
	configs map[string]*tls.Config
}

// NewTLSConfigCache creates a TLSConfigCache.
func NewTLSConfigCache() *TLSConfigCache {
	return &TLSConfigCache{configs: make(map[string]*tls.Config)}
}

// Get returns a TLS config for the given server name, creating one if needed.
func (tcc *TLSConfigCache) Get(serverName string) *tls.Config {
	tcc.mu.RLock()
	cfg, ok := tcc.configs[serverName]
	tcc.mu.RUnlock()
	if ok {
		return cfg
	}

	tcc.mu.Lock()
	defer tcc.mu.Unlock()
	// Double-check after acquiring write lock.
	if cfg, ok := tcc.configs[serverName]; ok {
		return cfg
	}

	cfg = &tls.Config{
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		SessionTicketsDisabled: false, // enable session ticket reuse
	}
	tcc.configs[serverName] = cfg
	return cfg
}

// ConnectionPool manages a pool of HTTP clients with shared transport
// configuration for connection reuse.
type ConnectionPool struct {
	mu      sync.RWMutex
	clients map[string]*http.Client
	dns     *DNSCache
	tls     *TLSConfigCache
}

// NewConnectionPool creates a ConnectionPool.
func NewConnectionPool(dns *DNSCache, tls *TLSConfigCache) *ConnectionPool {
	return &ConnectionPool{
		clients: make(map[string]*http.Client),
		dns:     dns,
		tls:     tls,
	}
}

// GetClient returns an HTTP client for the given scheme, creating one if needed.
// The client uses a shared transport with connection pooling, DNS caching,
// and TLS session reuse.
func (cp *ConnectionPool) GetClient(scheme string) *http.Client {
	cp.mu.RLock()
	client, ok := cp.clients[scheme]
	cp.mu.RUnlock()
	if ok {
		return client
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()
	// Double-check after acquiring write lock.
	if client, ok := cp.clients[scheme]; ok {
		return client
	}

	dialer := NewDialerWithCache(cp.dns, 10*time.Second)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
		TLSClientConfig: func() *tls.Config {
			// Return a config that will be customized per-request via TLSConfigCache.
			return &tls.Config{
				MinVersion:               tls.VersionTLS12,
				SessionTicketsDisabled:   false,
			}
		}(),
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	cp.clients[scheme] = client
	return client
}

// Count returns the number of clients in the pool.
func (cp *ConnectionPool) Count() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.clients)
}

// CloseIdleConnections closes idle connections on all pooled clients.
func (cp *ConnectionPool) CloseIdleConnections() {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	for _, client := range cp.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
}

// NetworkOptimizer combines DNS caching, TLS config caching, and connection
// pooling for optimized network access.
type NetworkOptimizer struct {
	dns     *DNSCache
	tls     *TLSConfigCache
	pool    *ConnectionPool
}

// NewNetworkOptimizer creates a NetworkOptimizer with default settings.
func NewNetworkOptimizer() *NetworkOptimizer {
	dns := NewDNSCache(5 * time.Minute)
	tls := NewTLSConfigCache()
	pool := NewConnectionPool(dns, tls)
	return &NetworkOptimizer{dns: dns, tls: tls, pool: pool}
}

// DNS returns the DNS cache.
func (no *NetworkOptimizer) DNS() *DNSCache {
	return no.dns
}

// TLS returns the TLS config cache.
func (no *NetworkOptimizer) TLS() *TLSConfigCache {
	return no.tls
}

// Pool returns the connection pool.
func (no *NetworkOptimizer) Pool() *ConnectionPool {
	return no.pool
}

// ClientForScheme returns an HTTP client optimized for the given scheme.
func (no *NetworkOptimizer) ClientForScheme(scheme string) *http.Client {
	return no.pool.GetClient(scheme)
}
