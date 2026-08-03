package hotdog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	cascadia "github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// NodeDOM "DOM Node Struct definition"
type NodeDOM struct {
	Element string `json:"element"`
	Content string `json:"content"`

	Children    []*NodeDOM   `json:"children"`
	Attributes  []*Attribute `json:"attributes"`
	Style       *Stylesheet  `json:"style"`
	Parent      *NodeDOM     `json:"-"`
	FirstChild  *NodeDOM     `json:"-"`
	NextSibling *NodeDOM     `json:"-"`
	PrevSibling *NodeDOM     `json:"-"`
	RenderBox   *RenderBox   `json:"-"`

	NeedsReflow  bool `json:"-"`
	NeedsRepaint bool `json:"-"`

	Document *Document  `json:"-"`
	HTMLNode *html.Node `json:"-"` // Reference to original html.Node for CSS selector queries
}

func (node *NodeDOM) Print(d int) {
	spacing := strings.Repeat("-", d)
	fmt.Printf("|%s> %s [%s]\n", spacing, node.Element, node.Content)

	for _, child := range node.Children {
		child.Print(d + 1)
	}
}

func (node *NodeDOM) JSON() string {
	res, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(res)
}

func (node *NodeDOM) FindByXPath(xPath string) (*NodeDOM, error) {
	if node.GetXPath() == xPath {
		return node, nil
	}

	for _, child := range node.Children {
		foundChild, err := child.FindByXPath(xPath)
		if err != nil {
			var noChild NoSuchElementError
			if errors.As(err, &noChild) {
				// No child with that element name, continue in other branches of the element tree
				continue
			}

			// Some other error
			return nil, err
		}

		return foundChild, nil
	}

	return nil, NoSuchElementError(xPath)
}

func (node *NodeDOM) GetXPath() string {
	return getXPath(node)
}

func (node *NodeDOM) FindChildByName(childName string) (*NodeDOM, error) {
	if node.Element == childName {
		return node, nil
	}

	for _, child := range node.Children {
		foundChild, err := child.FindChildByName(childName)
		if err != nil {
			var noChild NoSuchElementError
			if errors.As(err, &noChild) {
				// No child with that element name, continue in other branches of the element tree
				continue
			}

			// Some other error
			return nil, err
		}

		return foundChild, nil
	}

	return nil, NoSuchElementError(childName)
}

func (node *NodeDOM) Attr(attrName string) string {
	for _, attribute := range node.Attributes {
		if attribute.Name == attrName {
			return attribute.Value
		}
	}

	return ""
}

func (node *NodeDOM) CalcPointIntersection(x, y float64) *NodeDOM {
	var intersectedNode *NodeDOM
	if x > float64(node.RenderBox.Left) &&
		x < float64(node.RenderBox.Left+node.RenderBox.Width) &&
		y > float64(node.RenderBox.Top) &&
		y < float64(node.RenderBox.Top+node.RenderBox.Height) {
		intersectedNode = node
	}

	for i := 0; i < len(node.Children); i++ {
		tempNode := node.Children[i].CalcPointIntersection(x, y)
		if tempNode != nil {
			intersectedNode = tempNode
		}
	}

	return intersectedNode
}

func (node NodeDOM) RequestRepaint() {
	node.NeedsRepaint = true

	for _, childNode := range node.Children {
		childNode.RequestRepaint()
	}
}

func (node NodeDOM) RequestReflow() {
	node.NeedsReflow = true

	for _, childNode := range node.Children {
		childNode.RequestReflow()
	}
}

// QuerySelector returns the first element that matches the given CSS selector.
func (node *NodeDOM) QuerySelector(selector string) *NodeDOM {
	if node.HTMLNode == nil || node.Document == nil || node.Document.HTMLRoot == nil {
		return nil
	}

	sel, err := cascadia.Compile(selector)
	if err != nil {
		return nil
	}

	matched := sel.MatchFirst(node.Document.HTMLRoot)
	if matched == nil {
		return nil
	}

	return node.findNodeDOMByHTMLNode(matched)
}

// QuerySelectorAll returns all elements that match the given CSS selector.
func (node *NodeDOM) QuerySelectorAll(selector string) []*NodeDOM {
	if node.HTMLNode == nil || node.Document == nil || node.Document.HTMLRoot == nil {
		return nil
	}

	sel, err := cascadia.Compile(selector)
	if err != nil {
		return nil
	}

	matched := sel.MatchAll(node.Document.HTMLRoot)
	results := make([]*NodeDOM, 0, len(matched))
	for _, m := range matched {
		if nd := node.findNodeDOMByHTMLNode(m); nd != nil {
			results = append(results, nd)
		}
	}
	return results
}

// GetElementById returns the element with the given ID.
func (node *NodeDOM) GetElementById(id string) *NodeDOM {
	return node.QuerySelector("#" + escapeCSSIdentifier(id))
}

// GetElementsByClassName returns all elements with the given class name.
func (node *NodeDOM) GetElementsByClassName(className string) []*NodeDOM {
	return node.QuerySelectorAll("." + escapeCSSIdentifier(className))
}

// GetElementsByTagName returns all elements with the given tag name.
func (node *NodeDOM) GetElementsByTagName(tagName string) []*NodeDOM {
	return node.QuerySelectorAll(tagName)
}

// findNodeDOMByHTMLNode finds the NodeDOM corresponding to the given html.Node.
// It searches from the current node's document root.
func (node *NodeDOM) findNodeDOMByHTMLNode(target *html.Node) *NodeDOM {
	if node.Document == nil || node.Document.DOM == nil {
		return nil
	}
	return node.Document.DOM.findByHTMLNode(target)
}

// findByHTMLNode recursively searches for a NodeDOM with the given html.Node.
func (node *NodeDOM) findByHTMLNode(target *html.Node) *NodeDOM {
	if node.HTMLNode == target {
		return node
	}
	for _, child := range node.Children {
		if found := child.findByHTMLNode(target); found != nil {
			return found
		}
	}
	return nil
}

// escapeCSSIdentifier escapes a string for use as a CSS identifier.
func escapeCSSIdentifier(s string) string {
	var result strings.Builder
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r >= 0x80 {
			// First character cannot be a digit or hyphen followed by digit
			if i == 0 && (r >= '0' && r <= '9') {
				result.WriteString("\\")
				result.WriteRune(r)
			} else if i == 0 && r == '-' && len(s) > 1 && s[1] >= '0' && s[1] <= '9' {
				result.WriteString("\\")
				result.WriteRune(r)
			} else {
				result.WriteRune(r)
			}
		} else {
			// Escape special characters
			result.WriteString("\\")
			result.WriteRune(r)
		}
	}
	return result.String()
}
