package ketchup

import (
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mayo "github.com/danfragoso/thdwb/mayo"

	"golang.org/x/net/html"
)

func buildNodeDOMFromHTML(node *html.Node, document *hotdog.Document, windowCtx *hotdog.WindowContext) *hotdog.NodeDOM {
	var element, content string
	var nodeType hotdog.NodeType

	nodeDOM := &hotdog.NodeDOM{}
	attributes := retrieveAttributes(node)

	// Extract iframe-specific attributes before processing children
	var iframeSrc string
	var sandboxFlags hotdog.SandboxFlags
	if node.Type == html.ElementNode && node.Data == "iframe" {
		for _, attr := range attributes {
			if attr.Name == "src" {
				iframeSrc = attr.Value
			} else if attr.Name == "sandbox" {
				sandboxFlags = hotdog.ParseSandboxAttribute(attr.Value)
			}
		}
		nodeDOM.IframeSrc = iframeSrc
		nodeDOM.SandboxFlags = sandboxFlags
	}

	children := retrieveChildren(node)
	var prevChild *hotdog.NodeDOM
	for i, child := range children {
		childDOM := buildNodeDOMFromHTML(child, document, windowCtx)
		childDOM.Parent = nodeDOM
		childDOM.PrevSibling = prevChild
		if prevChild != nil {
			prevChild.NextSibling = childDOM
		}
		if i == 0 {
			nodeDOM.FirstChild = childDOM
		}
		prevChild = childDOM

		nodeDOM.Children = append(
			nodeDOM.Children,
			childDOM,
		)
	}

	switch node.Type {
	case html.TextNode:
		nodeType = hotdog.NodeTypeText
		element = "html:text"
		content = node.Data
	case html.ElementNode:
		nodeType = hotdog.NodeTypeElement
		element = node.Data
	case html.DoctypeNode:
		nodeType = hotdog.NodeTypeDoctype
		element = "html:doctype"
	case html.RawNode:
		nodeType = hotdog.NodeTypeRaw
		element = "html:raw"
	case html.CommentNode:
		nodeType = hotdog.NodeTypeComment
		element = "html:comment"
		content = node.Data
	}

	nodeDOM.Type = nodeType
	nodeDOM.Element = element
	nodeDOM.Content = content
	nodeDOM.Attributes = attributes
	nodeDOM.Document = document
	nodeDOM.WindowCtx = windowCtx
	nodeDOM.NeedsReflow = true
	nodeDOM.NeedsRepaint = true
	nodeDOM.Style = mayo.GetElementStylesheet(element, attributes)
	nodeDOM.RenderBox = &hotdog.RenderBox{}
	nodeDOM.HTMLNode = node

	// Handle <style> elements - extract and parse CSS
	if node.Type == html.ElementNode && node.Data == "style" {
		cssContent := extractStyleContent(node)
		if cssContent != "" {
			styleElements := mayo.ParseStylesheet(cssContent)
			document.StyleSheets = append(document.StyleSheets, styleElements...)
		}
		// Style elements don't render
		nodeDOM.Style.Display = "none"
	}

	return nodeDOM
}

func extractStyleContent(node *html.Node) string {
	var content string
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			content += child.Data
		}
	}
	return content
}

func retrieveChildren(node *html.Node) []*html.Node {
	var children []*html.Node
	if node.FirstChild == nil {
		return children
	}

	child := node.FirstChild
	children = append(children, child)

	for child.NextSibling != nil {
		child = child.NextSibling
		children = append(children, child)
	}

	return children
}

func retrieveAttributes(node *html.Node) []*hotdog.Attribute {
	var attributes []*hotdog.Attribute
	for _, attribute := range node.Attr {
		attributes = append(attributes, &hotdog.Attribute{
			Name:  attribute.Key,
			Value: attribute.Val,
		})
	}

	return attributes
}
