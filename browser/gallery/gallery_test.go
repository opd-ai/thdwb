package gallery

import (
	"testing"

	extensions "github.com/danfragoso/thdwb/browser/extensions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogAddAndGetListing(t *testing.T) {
	c := NewCatalog()
	listing := &ExtensionListing{
		ID:          "test-ext",
		Name:        "Test Extension",
		Version:     "1.0.0",
		Description: "A test extension",
		Author:      "Test Author",
		Category:    CategoryProductivity,
		Rating:      4.5,
		Downloads:   1000,
		XPIURL:      "https://example.com/xpi/test.xpi",
		Permissions: []string{"tabs", "storage"},
	}
	c.AddListing(listing)

	got := c.GetListing("test-ext")
	require.NotNil(t, got)
	assert.Equal(t, "Test Extension", got.Name)
	assert.Equal(t, CategoryProductivity, got.Category)
	assert.Equal(t, 1000, got.Downloads)
}

func TestCatalogGetMissingListing(t *testing.T) {
	c := NewCatalog()
	assert.Nil(t, c.GetListing("nonexistent"))
}

func TestCatalogListAllSortedByDownloads(t *testing.T) {
	c := NewCatalog()
	c.AddListing(&ExtensionListing{ID: "a", Name: "A", Downloads: 100})
	c.AddListing(&ExtensionListing{ID: "b", Name: "B", Downloads: 500})
	c.AddListing(&ExtensionListing{ID: "c", Name: "C", Downloads: 200})

	all := c.ListAll()
	require.Len(t, all, 3)
	assert.Equal(t, "b", all[0].ID)
	assert.Equal(t, "c", all[1].ID)
	assert.Equal(t, "a", all[2].ID)
}

func TestCatalogListByCategory(t *testing.T) {
	c := NewCatalog()
	c.AddListing(&ExtensionListing{ID: "a", Name: "A", Category: CategoryPrivacy, Downloads: 100})
	c.AddListing(&ExtensionListing{ID: "b", Name: "B", Category: CategorySecurity, Downloads: 200})
	c.AddListing(&ExtensionListing{ID: "c", Name: "C", Category: CategoryPrivacy, Downloads: 300})

	privacy := c.ListByCategory(CategoryPrivacy)
	require.Len(t, privacy, 2)
	assert.Equal(t, "c", privacy[0].ID)
	assert.Equal(t, "a", privacy[1].ID)

	security := c.ListByCategory(CategorySecurity)
	require.Len(t, security, 1)
	assert.Equal(t, "b", security[0].ID)
}

func TestCatalogCategories(t *testing.T) {
	c := NewCatalog()
	c.AddListing(&ExtensionListing{ID: "a", Name: "A", Category: CategorySecurity})
	c.AddListing(&ExtensionListing{ID: "b", Name: "B", Category: CategoryPrivacy})
	c.AddListing(&ExtensionListing{ID: "c", Name: "C", Category: CategorySecurity})

	cats := c.Categories()
	require.Len(t, cats, 2)
	assert.Equal(t, CategoryPrivacy, cats[0])
	assert.Equal(t, CategorySecurity, cats[1])
}

func TestCatalogSearch(t *testing.T) {
	c := NewCatalog()
	c.AddListing(&ExtensionListing{ID: "a", Name: "AdBlock Plus", Description: "Blocks ads"})
	c.AddListing(&ExtensionListing{ID: "b", Name: "Dark Reader", Description: "Dark mode"})
	c.AddListing(&ExtensionListing{ID: "c", Name: "LastPass", Description: "Password manager"})

	results := c.Search("block")
	require.Len(t, results, 1)
	assert.Equal(t, "a", results[0].ID)

	results = c.Search("reader")
	require.Len(t, results, 1)
	assert.Equal(t, "b", results[0].ID)

	results = c.Search("nonexistent")
	assert.Empty(t, results)
}

func TestCatalogCount(t *testing.T) {
	c := NewCatalog()
	assert.Equal(t, 0, c.Count())

	c.AddListing(&ExtensionListing{ID: "a", Name: "A"})
	assert.Equal(t, 1, c.Count())

	c.AddListing(&ExtensionListing{ID: "b", Name: "B"})
	assert.Equal(t, 2, c.Count())
}

func TestDefaultCatalogHasListings(t *testing.T) {
	c := DefaultCatalog()
	assert.Greater(t, c.Count(), 0)

	ublock := c.GetListing("u-block-origin")
	require.NotNil(t, ublock)
	assert.Equal(t, "uBlock Origin", ublock.Name)
	assert.Equal(t, CategoryPrivacy, ublock.Category)
	assert.Greater(t, ublock.Downloads, 0)
}

func TestGalleryInstallerInstallFromGalleryMissingListing(t *testing.T) {
	c := NewCatalog()
	extInstaller := extensions.NewInstaller(extensions.NewManager(""), "")
	gi := NewInstaller(c, extInstaller, "/tmp")

	_, err := gi.InstallFromGallery("nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in gallery")
}

func TestCategoryConstants(t *testing.T) {
	assert.Equal(t, "productivity", string(CategoryProductivity))
	assert.Equal(t, "privacy", string(CategoryPrivacy))
	assert.Equal(t, "security", string(CategorySecurity))
	assert.Equal(t, "shopping", string(CategoryShopping))
	assert.Equal(t, "social", string(CategorySocial))
	assert.Equal(t, "developer", string(CategoryDeveloper))
	assert.Equal(t, "accessibility", string(CategoryAccessibility))
	assert.Equal(t, "other", string(CategoryOther))
}
