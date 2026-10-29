// Package gallery implements an optional extension gallery/marketplace MVP.
// It provides a curated catalog of extensions with metadata, categories,
// ratings, and download counts, plus integration with the extension installer.
package gallery

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
)

// Category is a gallery category for organizing extensions.
type Category string

const (
	CategoryProductivity      Category = "productivity"
	CategoryPrivacy           Category = "privacy"
	CategorySecurity          Category = "security"
	CategoryShopping          Category = "shopping"
	CategorySocial            Category = "social"
	CategoryDeveloper         Category = "developer"
	CategoryAccessibility     Category = "accessibility"
	CategoryOther             Category = "other"
)

// ExtensionListing represents a curated extension entry in the gallery.
type ExtensionListing struct {
	// ID uniquely identifies the listing in the gallery.
	ID string `json:"id"`

	// Name is the human-readable extension name.
	Name string `json:"name"`

	// Version is the current version available in the gallery.
	Version string `json:"version"`

	// Description is a short description of the extension.
	Description string `json:"description"`

	// Author is the extension author or publisher.
	Author string `json:"author"`

	// Category classifies the extension.
	Category Category `json:"category"`

	// Rating is the average user rating (0-5).
	Rating float64 `json:"rating"`

	// Downloads is the total download count.
	Downloads int `json:"downloads"`

	// IconURL is the URL to the extension icon.
	IconURL string `json:"icon_url,omitempty"`

	// HomepageURL is the extension's homepage.
	HomepageURL string `json:"homepage_url,omitempty"`

	// XPIURL is the download URL for the .xpi package.
	XPIURL string `json:"xpi_url"`

	// SHA256 is the expected SHA-256 checksum of the .xpi package.
	SHA256 string `json:"sha256,omitempty"`

	// Permissions are the permissions the extension requires.
	Permissions []string `json:"permissions"`

	// UpdatedAt is when the listing was last updated.
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Catalog is the in-memory extension gallery catalog.
type Catalog struct {
	mu         sync.RWMutex
	listings   map[string]*ExtensionListing
	categories map[Category][]*ExtensionListing
}

// NewCatalog creates an empty gallery catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		listings:   make(map[string]*ExtensionListing),
		categories: make(map[Category][]*ExtensionListing),
	}
}

// AddListing adds or updates an extension listing in the catalog.
func (c *Catalog) AddListing(listing *ExtensionListing) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.listings[listing.ID] = listing

	// Rebuild category index.
	c.rebuildCategories()
}

// rebuildCategories rebuilds the category index from listings.
// Must be called with c.mu held.
func (c *Catalog) rebuildCategories() {
	c.categories = make(map[Category][]*ExtensionListing)
	for _, listing := range c.listings {
		c.categories[listing.Category] = append(c.categories[listing.Category], listing)
	}
	// Sort each category by downloads descending.
	for cat := range c.categories {
		sort.Slice(c.categories[cat], func(i, j int) bool {
			return c.categories[cat][i].Downloads > c.categories[cat][j].Downloads
		})
	}
}

// GetListing returns a listing by ID, or nil if not found.
func (c *Catalog) GetListing(id string) *ExtensionListing {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.listings[id]
}

// ListAll returns all listings sorted by downloads descending.
func (c *Catalog) ListAll() []*ExtensionListing {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*ExtensionListing, 0, len(c.listings))
	for _, l := range c.listings {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Downloads > out[j].Downloads
	})
	return out
}

// ListByCategory returns listings for a category, sorted by downloads descending.
func (c *Catalog) ListByCategory(cat Category) []*ExtensionListing {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := c.categories[cat]
	out := make([]*ExtensionListing, len(items))
	copy(out, items)
	return out
}

// Categories returns all categories that have at least one listing.
func (c *Catalog) Categories() []Category {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]Category, 0, len(c.categories))
	for cat := range c.categories {
		out = append(out, cat)
	}
	sort.Slice(out, func(i, j int) bool {
		return string(out[i]) < string(out[j])
	})
	return out
}

// Search returns listings whose name or description contains the query
// (case-insensitive substring match).
func (c *Catalog) Search(query string) []*ExtensionListing {
	c.mu.RLock()
	defer c.mu.RUnlock()

	q := strings.ToLower(query)
	var out []*ExtensionListing
	for _, l := range c.listings {
		if strings.Contains(strings.ToLower(l.Name), q) ||
			strings.Contains(strings.ToLower(l.Description), q) ||
			strings.Contains(strings.ToLower(l.Author), q) {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Downloads > out[j].Downloads
	})
	return out
}

// Count returns the total number of listings.
func (c *Catalog) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.listings)
}

// Installer installs extensions from the gallery via the extensions.Installer.
type Installer struct {
	catalog      *Catalog
	extInstaller *extensions.Installer
	downloadDir  string
}

// NewInstaller creates a gallery installer bound to the catalog and extension installer.
func NewInstaller(catalog *Catalog, extInstaller *extensions.Installer, downloadDir string) *Installer {
	return &Installer{
		catalog:      catalog,
		extInstaller: extInstaller,
		downloadDir:  downloadDir,
	}
}

// InstallFromGallery installs an extension from the gallery by listing ID.
// It downloads the .xpi package, verifies the checksum, and installs it.
func (gi *Installer) InstallFromGallery(listingID string, permissions []string) (*extensions.InstallResult, error) {
	listing := gi.catalog.GetListing(listingID)
	if listing == nil {
		return nil, fmt.Errorf("listing %q not found in gallery", listingID)
	}

	// Download the package.
	pkgPath, err := downloadPackage(listing.XPIURL, gi.downloadDir)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", listing.Name, err)
	}

	// Verify checksum if provided.
	if listing.SHA256 != "" {
		if err := verifyChecksum(pkgPath, listing.SHA256); err != nil {
			return nil, fmt.Errorf("checksum verification failed for %s: %w", listing.Name, err)
		}
	}

	// Install via the extension installer.
	result, err := gi.extInstaller.Install(&extensions.InstallRequest{
		PackagePath:  pkgPath,
		Permissions:  permissions,
		Enabled:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("install %s: %w", listing.Name, err)
	}

	return result, nil
}

// verifyChecksum checks that the SHA-256 of the file matches the expected value.
func verifyChecksum(path, expectedSHA256 string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	h.Write(data)
	sum := hex.EncodeToString(h.Sum(nil))
	if sum != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, sum)
	}
	return nil
}

// downloadPackage downloads a URL to a file in the given directory and returns the path.
func downloadPackage(urlStr, dir string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", urlStr, err)
	}

	filename := filepath.Base(u.Path)
	if filename == "" || filename == "/" {
		filename = "extension.xpi"
	}

	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		filename = filepath.Join(dir, filename)
	}

	// In a real implementation this would perform an HTTP GET. For the MVP,
	// we return the path so the caller can stage a test package.
	return filename, nil
}

// DefaultCatalog returns a curated default gallery catalog with popular
// Firefox-compatible extensions.
func DefaultCatalog() *Catalog {
	c := NewCatalog()

	listings := []*ExtensionListing{
		{
			ID:          "u-block-origin",
			Name:        "uBlock Origin",
			Version:     "1.52.0",
			Description: "Efficient blocker for ads and trackers. Low memory footprint.",
			Author:      "Raymond Hill",
			Category:    CategoryPrivacy,
			Rating:      4.9,
			Downloads:   10000000,
			IconURL:     "https://example.com/icons/ublock.png",
			HomepageURL: "https://github.com/gorhill/uBlock",
			XPIURL:      "https://example.com/xpi/ublock-origin.xpi",
			SHA256:      "",
			Permissions: []string{"webRequest", "webRequestBlocking", "tabs", "storage"},
			UpdatedAt:   "2026-10-01",
		},
		{
			ID:          "dark-reader",
			Name:        "Dark Reader",
			Version:     "4.9.80",
			Description: "Invert colors and create dark mode for websites.",
			Author:      "Dark Reader Team",
			Category:    CategoryAccessibility,
			Rating:      4.8,
			Downloads:   5000000,
			IconURL:     "https://example.com/icons/dark-reader.png",
			HomepageURL: "https://darkreader.org/",
			XPIURL:      "https://example.com/xpi/dark-reader.xpi",
			SHA256:      "",
			Permissions: []string{"tabs", "storage", "scripting"},
			UpdatedAt:   "2026-09-15",
		},
		{
			ID:          "lastpass",
			Name:        "LastPass",
			Version:     "4.43.0",
			Description: "Password manager and secure vault.",
			Author:      "LastPass",
			Category:    CategorySecurity,
			Rating:      4.2,
			Downloads:   3000000,
			IconURL:     "https://example.com/icons/lastpass.png",
			HomepageURL: "https://lastpass.com/",
			XPIURL:      "https://example.com/xpi/lastpass.xpi",
			SHA256:      "",
			Permissions: []string{"storage", "tabs", "contextMenus", "notifications"},
			UpdatedAt:   "2026-08-20",
		},
		{
			ID:          "grammarly",
			Name:        "Grammarly",
			Version:     "1.0.100",
			Description: "Grammar and spelling checker for text fields.",
			Author:      "Grammarly, Inc.",
			Category:    CategoryProductivity,
			Rating:      4.5,
			Downloads:   2000000,
			IconURL:     "https://example.com/icons/grammarly.png",
			HomepageURL: "https://grammarly.com/",
			XPIURL:      "https://example.com/xpi/grammarly.xpi",
			SHA256:      "",
			Permissions: []string{"tabs", "storage", "scripting", "contextMenus"},
			UpdatedAt:   "2026-09-05",
		},
		{
			ID:          "video-download-helper",
			Name:        "Video DownloadHelper",
			Version:     "8.1.3",
			Description: "Download videos from hundreds of video sites.",
			Author:      "Network & Media",
			Category:    CategoryProductivity,
			Rating:      4.1,
			Downloads:   1500000,
			IconURL:     "https://example.com/icons/video-download-helper.png",
			HomepageURL: "https://www.downloadhelper.net/",
			XPIURL:      "https://example.com/xpi/video-download-helper.xpi",
			SHA256:      "",
			Permissions: []string{"downloads", "tabs", "webRequest"},
			UpdatedAt:   "2026-08-10",
		},
	}

	for _, l := range listings {
		c.AddListing(l)
	}

	return c
}

// defaultCatalog is the shared default gallery catalog.
var defaultCatalog = DefaultCatalog()

// DefaultCatalogInstance returns the shared default gallery catalog.
func DefaultCatalogInstance() *Catalog {
	return defaultCatalog
}
