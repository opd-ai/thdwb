package shell

import (
	"net/url"
)

// AddressBar represents the browser's URL bar.
type AddressBar struct {
	URL string
}

// NewAddressBar creates a new address bar.
func NewAddressBar() *AddressBar {
	return &AddressBar{}
}

// SetURL sets the URL in the address bar.
func (ab *AddressBar) SetURL(u string) {
	ab.URL = u
}

// GetURL returns the current URL.
func (ab *AddressBar) GetURL() string {
	return ab.URL
}

// ParseURL parses the current URL.
func (ab *AddressBar) ParseURL() (*url.URL, error) {
	return url.Parse(ab.URL)
}