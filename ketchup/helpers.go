package ketchup

import (
	hotdog "github.com/danfragoso/thdwb/hotdog"
	mayo "github.com/danfragoso/thdwb/mayo"

	"golang.org/x/net/html"
)

func buildNodeDOMFromHTML(node *html.Node, document *hotdog.Document) *hotdog.NodeDOM {
	var element, content string

	nodeDOM := &hotdog.NodeDOM{}
	attributes := retrieveAttributes(node)

	children := retrieveChildren(node)
	var prevChild *hotdog.NodeDOM
	for i, child := range children {
		childDOM := buildNodeDOMFromHTML(child, document)
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
		element = "html:text"
		content = node.Data
	case html.ElementNode:
		element = node.Data
	case html.DoctypeNode:
		element = "html:doctype"
	case html.RawNode:
		element = "html:raw"
	case html.CommentNode:
		element = "html:comment"
		content = node.Data
	}

	nodeDOM.Element = element
	nodeDOM.Content = content
	nodeDOM.Attributes = attributes
	nodeDOM.Document = document
	nodeDOM.NeedsReflow = true
	nodeDOM.NeedsRepaint = true
	nodeDOM.Style = mayo.GetElementStylesheet(element, attributes)
	nodeDOM.RenderBox = &hotdog.RenderBox{}
	nodeDOM.HTMLNode = node

	return nodeDOM
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
