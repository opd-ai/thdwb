package bun

import (
	"testing"

	hotdog "github.com/danfragoso/thdwb/hotdog"
)

func BenchmarkGetPageTitle(b *testing.B) {
	root := &hotdog.NodeDOM{
		Element: "html",
		Children: []*hotdog.NodeDOM{
			{
				Element: "head",
				Children: []*hotdog.NodeDOM{
					{
						Element: "title",
						Content: "Test Page",
					},
				},
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetPageTitle(root)
	}
}

func BenchmarkParseFlexValue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseFlexValue("100px")
		parseFlexValue("50%")
		parseFlexValue("auto")
	}
}
