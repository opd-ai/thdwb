package ketchup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractAttributesKeepsEqualsInValue (audit O-17) covers the regex
// tokenizer's attribute split: values containing "=" (query strings in hrefs,
// data attributes) must survive intact. The old strings.Split on every "="
// truncated the value at the first embedded equals sign.
//
// Note: this tokenizer path is dead/test-only -- the live document parse is
// ParseHTMLDocument via x/net/html -- but extractAttributes is still exercised
// by tests and the fix prevents a latent cut if the path is ever reused.
func TestExtractAttributesKeepsEqualsInValue(t *testing.T) {
	attrs := extractAttributes(`href="https://x.example/a?b=c&d=e" class="c1"`)
	assert.Len(t, attrs, 2)

	var href string
	for _, a := range attrs {
		if a.Name == "href" {
			href = a.Value
		}
	}
	assert.Equal(t, "https://x.example/a?b=c&d=e", href, "value must keep embedded '=' chars")
}
