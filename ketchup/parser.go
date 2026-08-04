package ketchup

import (
	"strings"

	hotdog "github.com/danfragoso/thdwb/hotdog"
	mayo "github.com/danfragoso/thdwb/mayo"

	"golang.org/x/net/html"
)

func ParseHTMLDocument(document string, windowCtx *hotdog.WindowContext) *hotdog.Document {
	parsedDoc, err := html.Parse(strings.NewReader(document))
	if err != nil {
		panic(err)
	}

	HTMLDocument := &hotdog.Document{}
	HTMLDocument.RawDocument = document
	HTMLDocument.HTMLRoot = parsedDoc

	HTMLDocument.DOM = buildNodeDOMFromHTML(parsedDoc, HTMLDocument, windowCtx)

	// Apply CSS inheritance after all stylesheets are processed
	mayo.ApplyInheritance(HTMLDocument.DOM)

	return HTMLDocument
}
