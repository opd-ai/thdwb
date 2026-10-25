package ketchup

import (
	"regexp"
	"strings"

	hotdog "github.com/danfragoso/thdwb/hotdog"
	mayo "github.com/danfragoso/thdwb/mayo"
)

// Regular expressions for HTML tokenization.
// These patterns identify tags, attributes, and content in HTML text.
// Note: This is a simplified parser for learning purposes. A full HTML5 parser
// would use a state machine per the HTML spec.
// See: https://html.spec.whatwg.org/multipage/parsing.html
// See: https://searchfox.org/mozilla-central/source/parser/html/HTMLTokenizer.cpp
var (
	// xmlTag matches any tag: <tag>, </tag>, <tag/>, <tag attr="val">
	xmlTag = regexp.MustCompile(`(\<.+?\>)|(\<//?\w+\>\\?)`)

	// clTag matches closing tags: </tag>
	clTag = regexp.MustCompile(`\<\/\w+\>`)

	// selfClosingTag matches self-closing tags: <tag/>
	selfClosingTag = regexp.MustCompile(`\<.+\/\>`)

	// tagContent matches content between opening and closing tags
	tagContent = regexp.MustCompile(`(.+?)\<\/`)

	// tagName matches the tag name in an opening tag: <tagname
	tagName = regexp.MustCompile(`(\<\w+)`)

	// attr matches attributes: name="value"
	attr = regexp.MustCompile(`\w+=".+?"`)
)

// extractAttributes parses attributes from a tag string.
// Returns a slice of Attribute pointers.
// Example: 'class="foo" id="bar"' -> [{Name: "class", Value: "foo"}, {Name: "id", Value: "bar"}]
func extractAttributes(tag string) []*hotdog.Attribute {
	rawAttrArray := attr.FindAllString(tag, -1)
	elementAttrs := []*hotdog.Attribute{}

	for i := 0; i < len(rawAttrArray); i++ {
		attrStringSlice := strings.Split(rawAttrArray[i], "=")
		attr := &hotdog.Attribute{
			Name:  attrStringSlice[0],
			Value: strings.Trim(attrStringSlice[1], "\""),
		}

		elementAttrs = append(elementAttrs, attr)
	}

	return elementAttrs
}

// isVoidElement checks if an HTML element is a void element (self-closing).
// Void elements cannot have children and don't have closing tags.
// List matches HTML5 void elements per spec.
// See: https://developer.mozilla.org/en-US/docs/Glossary/Void_element
// See: https://searchfox.org/mozilla-central/source/parser/html/HTMLTreeBuilder.cpp#1200
func isVoidElement(tagName string) bool {
	var isVoid bool
	switch tagName {
	case "area",
		"base",
		"br",
		"col",
		"command",
		"embed",
		"hr",
		"img",
		"input",
		"keygen",
		"link",
		"meta",
		"param",
		"source",
		"track",
		"wbr":
		isVoid = true
	default:
		isVoid = false
	}

	return isVoid
}

// CreateHTMLBase builds the minimal HTML document structure:
// <html><head></head><body></body></html>
// Creates the root html, head, and body nodes with default styles.
// Used as fallback when parsing malformed HTML or for plain text documents.
// Parameters:
//
//	windowCtx - the window context for the new document
//
// Returns the root html NodeDOM.
func CreateHTMLBase(windowCtx *hotdog.WindowContext) *hotdog.NodeDOM {
	html := &hotdog.NodeDOM{
		Element:      "html",
		Type:         hotdog.NodeTypeElement,
		NeedsReflow:  true,
		NeedsRepaint: true,
		Style:        mayo.GetElementStylesheet("html", []*hotdog.Attribute{}),
		RenderBox:    &hotdog.RenderBox{},
		WindowCtx:    windowCtx,
	}

	head := &hotdog.NodeDOM{
		Element:      "head",
		Type:         hotdog.NodeTypeElement,
		NeedsReflow:  true,
		NeedsRepaint: true,
		Style:        mayo.GetElementStylesheet("head", []*hotdog.Attribute{}),
		RenderBox:    &hotdog.RenderBox{},
		Parent:       html,
		WindowCtx:    windowCtx,
	}

	body := &hotdog.NodeDOM{
		Element:      "body",
		Type:         hotdog.NodeTypeElement,
		NeedsReflow:  true,
		NeedsRepaint: true,
		Style:        mayo.GetElementStylesheet("body", []*hotdog.Attribute{}),
		RenderBox:    &hotdog.RenderBox{},
		Parent:       html,
		WindowCtx:    windowCtx,
	}

	html.Children = append(html.Children, head, body)
	return html
}

// ParsePlainText creates a minimal document for plain text (no HTML tags).
// Wraps each line in <p> tags inside a basic HTML structure.
// Used for text files, error pages, and non-HTML content.
// Parameters:
//
//	document - plain text content
//	windowCtx - window context for the new document
//
// Returns a Document with a basic DOM tree.
func ParsePlainText(document string, windowCtx *hotdog.WindowContext) *hotdog.Document {
	documentTitle := "Text Document"
	textDocument := &hotdog.Document{
		Title: documentTitle,

		RawDocument: document,
		DOM: &hotdog.NodeDOM{
			Element:      "html",
			Type:         hotdog.NodeTypeElement,
			NeedsReflow:  true,
			NeedsRepaint: true,
			Style:        mayo.GetElementStylesheet("html", []*hotdog.Attribute{}),
			RenderBox:    &hotdog.RenderBox{},
			WindowCtx:    windowCtx,
		},
	}

	textDocument.DOM.Document = textDocument
	textDocument.DOM.Children = []*hotdog.NodeDOM{
		{
			Element:   "head",
			Type:      hotdog.NodeTypeElement,
			Document:  textDocument,
			Style:     mayo.GetElementStylesheet("head", []*hotdog.Attribute{}),
			RenderBox: &hotdog.RenderBox{},
			Parent:    textDocument.DOM,
			WindowCtx: windowCtx,
		},
		{
			Element:      "body",
			Type:         hotdog.NodeTypeElement,
			NeedsReflow:  true,
			NeedsRepaint: true,
			Style:        mayo.GetElementStylesheet("body", []*hotdog.Attribute{}),
			RenderBox:    &hotdog.RenderBox{},
			Document:     textDocument,
			Parent:       textDocument.DOM,
			WindowCtx:    windowCtx,
		},
	}

	documentLines := strings.Split(document, "\n")
	body, _ := textDocument.DOM.FindChildByName("body")
	for _, line := range documentLines {
		body.Children = append(body.Children, &hotdog.NodeDOM{
			Element:   "p",
			Type:      hotdog.NodeTypeElement,
			Content:   line,
			RenderBox: &hotdog.RenderBox{},
			Style:     mayo.GetElementStylesheet("p", []*hotdog.Attribute{}),
			Parent:    body,
			WindowCtx: windowCtx,
		})
	}

	return textDocument
}

// ParseHTML parses a full HTML document string into a Document.
// Uses the custom ketchup parser (regex-based, simplified).
// Implements basic tokenization and tree construction.
// Does NOT fully implement HTML5 parsing algorithm (no foster parenting, etc.).
// Parameters:
//
//	document - HTML source text
//	windowCtx - window context for the new document
//
// Returns a Document with populated DOM tree and applied CSS inheritance.
// See: https://html.spec.whatwg.org/multipage/parsing.html
// See: https://searchfox.org/mozilla-central/source/parser/html
func ParseHTML(document string, windowCtx *hotdog.WindowContext) *hotdog.Document {
	HTMLDocument := &hotdog.Document{}

	HTMLDocument.RawDocument = document
	lastNode := HTMLDocument.DOM
	parseDocument := xmlTag.MatchString(document)
	document = strings.ReplaceAll(document, "\n", "")

	for parseDocument {
		var currentNode *hotdog.NodeDOM

		currentTag := xmlTag.FindString(document)
		currentTagIndex := xmlTag.FindStringIndex(document)

		if string(currentTag[1]) == "!" {
			document = strings.Replace(document, currentTag, "", 1)
		} else {
			if clTag.MatchString(currentTag) {
				contentStringMatch := tagContent.FindStringSubmatch(document)
				contentString := ""

				if len(contentStringMatch) > 1 {
					contentString = contentStringMatch[1]
				}

				if clTag.MatchString(contentString) {
					lastNode.Content = ""
				} else {
					if lastNode != nil {
						lastNode.Content = strings.TrimSpace(contentString)
					}
				}

				if lastNode.Parent != nil {
					lastNode = lastNode.Parent
				}
			} else {
				currentTagName := strings.Trim(tagName.FindString(currentTag), "<")

				extractedAttributes := extractAttributes(currentTag)
				elementStylesheet := mayo.GetElementStylesheet(currentTagName, extractedAttributes)

				if lastNode == nil {
					HTMLDocument.DOM = CreateHTMLBase(windowCtx)
					lastNode = HTMLDocument.DOM.Children[1]
				}

				currentNode = &hotdog.NodeDOM{
					Element:    currentTagName,
					Type:       hotdog.NodeTypeElement,
					Content:    "",
					Children:   []*hotdog.NodeDOM{},
					Attributes: extractedAttributes,
					Style:      elementStylesheet,
					Parent:     lastNode,

					NeedsReflow:  true,
					NeedsRepaint: true,
					RenderBox:    &hotdog.RenderBox{},

					Document:  HTMLDocument,
					WindowCtx: windowCtx,
				}

				if currentTagName == "html" {
					HTMLDocument.DOM = currentNode
					lastNode = HTMLDocument.DOM
				} else {
					lastNode.Children = append(lastNode.Children, currentNode)

					if !isVoidElement(currentTagName) {
						lastNode = currentNode
					}
				}
			}

			document = document[currentTagIndex[1]:]
		}

		if !xmlTag.MatchString(document) {
			parseDocument = false
		}
	}

	// Apply CSS inheritance after all stylesheets are processed
	mayo.ApplyInheritance(HTMLDocument.DOM)

	return HTMLDocument
}
