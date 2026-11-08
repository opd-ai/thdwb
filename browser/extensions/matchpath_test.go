package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMatchPathWildcardBoundary (audit O-27) verifies Chrome match-pattern
// path semantics: "/foo/*" matches "/foo" and "/foo/bar" but NOT "/foobar".
// The old code used a bare strings.HasPrefix after stripping "/*", so
// "/foobar" was wrongly matched and content scripts leaked past the declared
// boundary.
func TestMatchPathWildcardBoundary(t *testing.T) {
	assert.True(t, matchPath("/foo/*", "/foo"))
	assert.True(t, matchPath("/foo/*", "/foo/"))
	assert.True(t, matchPath("/foo/*", "/foo/bar"))
	assert.True(t, matchPath("/foo/*", "/foo/bar/baz"))
	assert.False(t, matchPath("/foo/*", "/foobar"))
	assert.False(t, matchPath("/foo/*", "/foobar/baz"))
	assert.False(t, matchPath("/foo/*", "/other"))
	assert.True(t, matchPath("/*", "/anything"))
	assert.True(t, matchPath("/", "/anything"))
	assert.True(t, matchPath("", "/anything"))
	assert.True(t, matchPath("/exact", "/exact"))
	assert.False(t, matchPath("/exact", "/other"))
}
