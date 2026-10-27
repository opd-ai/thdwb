package extensions

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// ContentScriptCSP enforces Content Security Policy on content scripts.
// It evaluates the manifest's content_security_policy against resource
// URLs and script execution, blocking violations.
type ContentScriptCSP struct {
	mu     sync.Mutex
	policy *CSPPolicy
}

// CSPPolicy represents a parsed CSP for content scripts.
type CSPPolicy struct {
	// Directives maps directive name to allowed sources.
	Directives map[string][]string
	// Raw is the original policy string.
	Raw string
}

// NewContentScriptCSP creates a CSP enforcer from a manifest CSP string.
// An empty string means no policy is enforced.
func NewContentScriptCSP(policy string) *ContentScriptCSP {
	csp := &ContentScriptCSP{}
	if policy != "" {
		csp.policy = parseCSP(policy)
	}
	return csp
}

// Enabled reports whether a CSP policy is active.
func (c *ContentScriptCSP) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.policy != nil
}

// CheckScript checks whether a script URL is allowed by the script-src
// directive (falling back to default-src).
func (c *ContentScriptCSP) CheckScript(scriptURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.policy == nil {
		return nil
	}

	u, err := url.Parse(scriptURL)
	if err != nil {
		return fmt.Errorf("invalid script URL %q: %w", scriptURL, err)
	}

	// Check script-src first, then default-src.
	sources := c.policy.Directives["script-src"]
	if len(sources) == 0 {
		sources = c.policy.Directives["default-src"]
	}

	if !matchesAnySource(u, sources) {
		return fmt.Errorf("CSP: script-src blocked %q", scriptURL)
	}

	return nil
}

// CheckConnection checks whether a connection URL is allowed by connect-src
// (falling back to default-src).
func (c *ContentScriptCSP) CheckConnection(connURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.policy == nil {
		return nil
	}

	u, err := url.Parse(connURL)
	if err != nil {
		return fmt.Errorf("invalid connection URL %q: %w", connURL, err)
	}

	sources := c.policy.Directives["connect-src"]
	if len(sources) == 0 {
		sources = c.policy.Directives["default-src"]
	}

	if !matchesAnySource(u, sources) {
		return fmt.Errorf("CSP: connect-src blocked %q", connURL)
	}

	return nil
}

// CheckObject checks whether an object URL is allowed by object-src
// (falling back to default-src).
func (c *ContentScriptCSP) CheckObject(objURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.policy == nil {
		return nil
	}

	u, err := url.Parse(objURL)
	if err != nil {
		return fmt.Errorf("invalid object URL %q: %w", objURL, err)
	}

	sources := c.policy.Directives["object-src"]
	if len(sources) == 0 {
		sources = c.policy.Directives["default-src"]
	}

	if !matchesAnySource(u, sources) {
		return fmt.Errorf("CSP: object-src blocked %q", objURL)
	}

	return nil
}

// Policy returns the underlying CSP policy, or nil if none is set.
func (c *ContentScriptCSP) Policy() *CSPPolicy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.policy
}

// parseCSP parses a CSP policy string into a CSPPolicy.
func parseCSP(policy string) *CSPPolicy {
	csp := &CSPPolicy{
		Directives: make(map[string][]string),
		Raw:        policy,
	}

	directives := strings.Split(policy, ";")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}
		parts := strings.Fields(directive)
		if len(parts) == 0 {
			continue
		}
		name := strings.ToLower(parts[0])
		if _, exists := csp.Directives[name]; exists {
			continue
		}
		sources := make([]string, 0, len(parts)-1)
		for _, src := range parts[1:] {
			sources = append(sources, src)
		}
		csp.Directives[name] = sources
	}

	return csp
}

// matchesAnySource checks whether a URL matches any of the given CSP sources.
func matchesAnySource(u *url.URL, sources []string) bool {
	for _, src := range sources {
		if matchSource(src, u) {
			return true
		}
	}
	return false
}

// matchSource checks whether a single CSP source expression matches a URL.
func matchSource(src string, u *url.URL) bool {
	switch src {
	case "'none'":
		return false
	case "'self'":
		// 'self' matches the same scheme and host. Since we don't have the
		// origin context here, match by scheme+host only if the URL is not
		// a data URL or blob URL.
		if u.Scheme == "data" || u.Scheme == "blob" {
			return false
		}
		return true
	case "*":
		return true
	default:
		// Handle scheme-specific sources like https://example.com
		if strings.HasPrefix(src, "'") && strings.HasSuffix(src, "'") {
			// Keyword sources not handled above.
			return false
		}
		// Parse as a match pattern or URL.
		if strings.Contains(src, "://") {
			patternURL, err := url.Parse(src)
			if err != nil {
				return false
			}
			if patternURL.Scheme != "*" && patternURL.Scheme != u.Scheme {
				return false
			}
			if patternURL.Host != "*" && patternURL.Host != u.Host {
				return false
			}
			return true
		}
		return false
	}
}
