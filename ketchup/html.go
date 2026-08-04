package ketchup

import (
	"regexp"
	"strings"

	hotdog "github.com/danfragoso/thdwb/hotdog"
	mayo "github.com/danfragoso/thdwb/mayo"
)

var (
	xmlTag         = regexp.MustCompile(`(\<.+?\>)|(\<//?\w+\>\\?)`)
	clTag          = regexp.MustCompile(`\<\/\w+\>`)
	selfClosingTag = regexp.MustCompile(`\<.+\/\>`)
	tagContent     = regexp.MustCompile(`(.+?)\<\/`)
	tagName        = regexp.MustCompile(`(\<\w+)`)
	attr           = regexp.MustCompile(`\w+=".+?"`)
)

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

func ParseHTML(document string, windowCtx *hotdog.WindowContext) *hotdog.Document {
	HTMLDocument := &hotdog.Document{}

	HTMLDocument.RawDocument = document
	lastNode := HTMLDocument.DOM
	parseDocument := xmlTag.MatchString(document)
	document = strings.ReplaceAll(document, "\n", "")

	for parseDocument == true {
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

	return HTMLDocument
}
