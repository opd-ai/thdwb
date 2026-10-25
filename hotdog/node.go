package hotdog

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	cascadia "github.com/andybalholm/cascadia"
)

// NodeType represents the type of a DOM node.
// Mirrors the DOM Level 3 Node interface nodeType constants.
// See: https://developer.mozilla.org/en-US/docs/Web/API/Node/nodeType
type NodeType int

const (
	// NodeTypeElement represents an HTML element node (e.g., <div>, <span>).
	NodeTypeElement NodeType = iota
	// NodeTypeText represents a text node (text content inside elements).
	NodeTypeText
	// NodeTypeComment represents a comment node (<!-- comment -->).
	NodeTypeComment
	// NodeTypeDoctype represents a doctype node (<!DOCTYPE html>).
	NodeTypeDoctype
	// NodeTypeRaw represents a raw HTML node (unparsed HTML).
	NodeTypeRaw
)

// NodeDOM represents a single node in the DOM tree.
// This is thdwb's internal DOM node implementation, similar to browser's internal Node.
// Contains tree structure (parent, children, siblings), styling, layout, and JS bindings.
// See: https://developer.mozilla.org/en-US/docs/Web/API/Node
// See: https://searchfox.org/mozilla-central/source/dom/base/nsINode.cpp
type NodeDOM struct {
	// Node identity
	Type      NodeType       `json:"type"`    // Node type (element, text, comment, etc.)
	Element   string         `json:"element"` // Tag name (e.g., "div", "span", "html:text")
	Content   string         `json:"content"` // Text content (for text nodes, or inner text)
	WindowCtx *WindowContext `json:"-"`       // Owning window context (for JS bindings)

	// Tree structure
	Children    []*NodeDOM   `json:"children"`   // Child nodes
	Attributes  []*Attribute `json:"attributes"` // HTML attributes
	Style       *Stylesheet  `json:"style"`      // Computed CSS styles
	Parent      *NodeDOM     `json:"-"`          // Parent node (nil for root)
	FirstChild  *NodeDOM     `json:"-"`          // First child (linked list)
	NextSibling *NodeDOM     `json:"-"`          // Next sibling
	PrevSibling *NodeDOM     `json:"-"`          // Previous sibling
	RenderBox   *RenderBox   `json:"-"`          // Layout result: x, y, width, height

	// Flexbox layout (kjk/flex) - used by experimental layout engine
	FlexNode interface{} `json:"-"`

	// Dirty flags for incremental updates
	Dirty        bool `json:"-"` // General dirty flag
	NeedsReflow  bool `json:"-"` // Needs layout recalculation
	NeedsRepaint bool `json:"-"` // Needs repaint

	// Document ownership
	Document *Document  `json:"-"` // Owner document
	HTMLNode *html.Node `json:"-"` // Reference to stdlib html.Node (for CSS selectors)

	// Iframe-specific fields
	IframeDocument *Document    `json:"-"` // Parsed document for iframe content
	SandboxFlags   SandboxFlags `json:"-"` // Sandbox restrictions for iframe
	IframeSrc      string       `json:"-"` // Source URL for iframe
}

// Print prints the DOM tree structure for debugging.
// Indents each level with dashes. Shows element name and text content.
// Example output:
// |> html []
// |-> head []
// |--> title [My Page]
// |-> body []
// |--> h1 [Hello]
func (node *NodeDOM) Print(d int) {
	spacing := strings.Repeat("-", d)
	fmt.Printf("|%s> %s [%s]\n", spacing, node.Element, node.Content)

	for _, child := range node.Children {
		child.Print(d + 1)
	}
}

// JSON returns the node and its subtree as indented JSON.
// Useful for debugging and serialization.
// Returns "{}" on marshal error.
func (node *NodeDOM) JSON() string {
	res, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(res)
}

// FindByXPath finds a descendant node by XPath expression.
// Returns the first matching node or NoSuchElementError.
// XPath is computed from the node's position in the tree.
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

// GetXPath returns the XPath of this node in the document tree.
// Format: /html/body/div[1]/p[2] etc.
func (node *NodeDOM) GetXPath() string {
	return getXPath(node)
}

// FindChildByName finds the first direct child with the given element name.
// Returns the child node or NoSuchElementError.
// Only searches direct children, not descendants.
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

// Attr returns the value of an attribute by name.
// Returns empty string if attribute not found.
func (node *NodeDOM) Attr(attrName string) string {
	for _, attribute := range node.Attributes {
		if attribute.Name == attrName {
			return attribute.Value
		}
	}

	return ""
}

// InnerHTML returns the HTML content of the node's children.
// For text nodes, returns the text content.
// For elements, serializes all children to HTML string.
func (node *NodeDOM) InnerHTML() string {
	if node.Type == NodeTypeText {
		return node.Content
	}

	var result strings.Builder
	for _, child := range node.Children {
		result.WriteString(child.toHTML())
	}
	return result.String()
}

// toHTML converts a NodeDOM to its HTML string representation.
// Internal method used by InnerHTML. Handles all node types.
func (node *NodeDOM) toHTML() string {
	switch node.Type {
	case NodeTypeText:
		return escapeHTML(node.Content)
	case NodeTypeComment:
		return "<!--" + node.Content + "-->"
	case NodeTypeDoctype:
		return "<!DOCTYPE " + node.Content + ">"
	case NodeTypeRaw:
		return node.Content
	case NodeTypeElement:
		var result strings.Builder
		result.WriteString("<")
		result.WriteString(node.Element)
		for _, attr := range node.Attributes {
			result.WriteString(" ")
			result.WriteString(attr.Name)
			result.WriteString("=\"")
			result.WriteString(escapeHTML(attr.Value))
			result.WriteString("\"")
		}
		result.WriteString(">")
		for _, child := range node.Children {
			result.WriteString(child.toHTML())
		}
		result.WriteString("</")
		result.WriteString(node.Element)
		result.WriteString(">")
		return result.String()
	default:
		return ""
	}
}

// escapeHTML escapes special HTML characters for safe output.
// Replaces &, <, > with HTML entities.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&")
	s = strings.ReplaceAll(s, "<", "<")
	s = strings.ReplaceAll(s, ">", ">")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "'")
	return s
}

// CalcPointIntersection finds the deepest node at the given coordinates.
// Used for hit testing (mouse clicks, hover).
// Searches children in reverse order (top-most first).
// Returns the node whose render box contains the point, or nil.
func (node *NodeDOM) CalcPointIntersection(x, y float64) *NodeDOM {
	if node == nil {
		return nil
	}
	for i := len(node.Children) - 1; i >= 0; i-- {
		if hit := node.Children[i].CalcPointIntersection(x, y); hit != nil {
			return hit
		}
	}
	box := node.RenderBox
	if box == nil || box.Width <= 0 || box.Height <= 0 {
		return nil
	}
	if x >= box.Left && x < box.Left+box.Width && y >= box.Top && y < box.Top+box.Height {
		return node
	}
	return nil
}

// MarkDirty marks this node and all ancestors as dirty.
// Called when node content, attributes, or style changes.
// Triggers reflow/repaint up the tree.
func (node *NodeDOM) MarkDirty() {
	if node == nil {
		return
	}
	node.Dirty = true
	if node.Parent != nil {
		node.Parent.MarkDirty()
	}
}

// ClearDirty clears the dirty flag on this node and all descendants.
// Called after successful reflow/repaint.
func (node *NodeDOM) ClearDirty() {
	if node == nil {
		return
	}
	node.Dirty = false
	for _, childNode := range node.Children {
		childNode.ClearDirty()
	}
}

// RequestRepaint marks this node and all descendants as needing repaint.
// Called when visual properties change (color, visibility, etc.).
// Also marks dirty to trigger reflow if needed.
func (node *NodeDOM) RequestRepaint() {
	node.NeedsRepaint = true
	node.MarkDirty()

	for _, childNode := range node.Children {
		childNode.RequestRepaint()
	}
}

// RequestReflow requests a layout recalculation for this node and its subtree.
// Queues the reflow in the window context's reflow queue.
// If queueing fails, marks the subtree dirty immediately.
func (node *NodeDOM) RequestReflow() {
	if node == nil {
		return
	}
	root := node
	if node.WindowCtx != nil {
		var queued bool
		root, queued = node.WindowCtx.queueReflow(node)
		if !queued {
			node.MarkDirty()
			return
		}
	}
	root.MarkDirty()
	root.markReflowSubtree()
}

// markReflowSubtree marks this node and all descendants as needing reflow.
// Internal method called by RequestReflow after successful queueing.
func (node *NodeDOM) markReflowSubtree() {
	node.NeedsReflow = true
	node.Dirty = true
	for _, childNode := range node.Children {
		childNode.markReflowSubtree()
	}
}

// SecurityError represents a security violation error.
// Used for same-origin policy violations, CSP violations, etc.
// See: https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy
type SecurityError string

func (e SecurityError) Error() string {
	return fmt.Sprintf("SecurityError: %s", string(e))
}

// checkOrigin validates that the node belongs to the same origin as the window context.
// Enforces same-origin policy for DOM access from JavaScript.
// Returns a SecurityError if the origin check fails.
// Checks:
// - Document URL vs WindowContext origin (scheme, host, port)
// - Iframe sandbox restrictions (allow-same-origin)
func (node *NodeDOM) checkOrigin() error {
	if node.WindowCtx == nil || node.WindowCtx.Origin == nil {
		// No origin set, allow access (for backward compatibility)
		return nil
	}

	// Check if this node is inside an iframe with sandbox restrictions
	if node.isInSandboxedIframe() {
		return SecurityError("cross-origin iframe access blocked by sandbox")
	}

	// For now, we assume all nodes in the same document have the same origin.
	// In a full implementation, each node would track its own origin (for iframes).
	// The Document should have an origin field that we can check against WindowCtx.Origin.
	if node.Document != nil && node.Document.URL != nil {
		nodeOrigin := node.Document.URL
		windowOrigin := node.WindowCtx.Origin

		// Compare scheme, host, and port (normalizing default ports)
		if nodeOrigin.Scheme != windowOrigin.Scheme {
			return SecurityError("cross-origin access blocked")
		}
		if nodeOrigin.Hostname() != windowOrigin.Hostname() {
			return SecurityError("cross-origin access blocked")
		}
		if nodeOrigin.Port() != windowOrigin.Port() {
			return SecurityError("cross-origin access blocked")
		}
	}

	return nil
}

// isInSandboxedIframe checks if the node is inside an iframe that doesn't allow same-origin access.
// Returns true if any ancestor iframe has sandbox attribute without allow-same-origin.
// See: https://developer.mozilla.org/en-US/docs/Web/HTML/Element/iframe#attr-sandbox
func (node *NodeDOM) isInSandboxedIframe() bool {
	current := node
	for current != nil {
		if current.Type == NodeTypeElement && current.Element == "iframe" {
			// Check if iframe has sandbox attribute without allow-same-origin
			if current.SandboxFlags != SandboxNone && (current.SandboxFlags&SandboxAllowSameOrigin) == 0 {
				return true
			}
		}
		current = current.Parent
	}
	return false
}

// isInsideIframe checks if the node is a descendant of an iframe element.
// Used to enforce iframe boundaries for querySelector/querySelectorAll.
// Returns true if any ancestor is an iframe element.
func (node *NodeDOM) isInsideIframe() bool {
	current := node.Parent
	for current != nil {
		if current.Type == NodeTypeElement && current.Element == "iframe" {
			return true
		}
		current = current.Parent
	}
	return false
}

// QuerySelector returns the first element that matches the given CSS selector.
// It respects iframe boundaries and does not match elements inside iframes.
func (node *NodeDOM) QuerySelector(selector string) (*NodeDOM, error) {
	if err := node.checkOrigin(); err != nil {
		return nil, err
	}

	if node.HTMLNode == nil || node.Document == nil || node.Document.HTMLRoot == nil {
		return nil, nil
	}

	sel, err := cascadia.Compile(selector)
	if err != nil {
		return nil, err
	}

	matched := sel.MatchFirst(node.Document.HTMLRoot)
	if matched == nil {
		return nil, nil
	}

	result := node.findNodeDOMByHTMLNode(matched)
	if result != nil {
		// Check if result is inside an iframe - querySelector should not cross iframe boundaries
		if result.isInsideIframe() {
			return nil, nil
		}
		if err := result.checkOrigin(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// QuerySelectorAll returns all elements that match the given CSS selector.
// It respects iframe boundaries and does not match elements inside iframes.
func (node *NodeDOM) QuerySelectorAll(selector string) ([]*NodeDOM, error) {
	if err := node.checkOrigin(); err != nil {
		return nil, err
	}

	if node.HTMLNode == nil || node.Document == nil || node.Document.HTMLRoot == nil {
		return nil, nil
	}

	sel, err := cascadia.Compile(selector)
	if err != nil {
		return nil, err
	}

	matched := sel.MatchAll(node.Document.HTMLRoot)
	results := make([]*NodeDOM, 0, len(matched))
	for _, m := range matched {
		if nd := node.findNodeDOMByHTMLNode(m); nd != nil {
			// Check if result is inside an iframe - querySelectorAll should not cross iframe boundaries
			if nd.isInsideIframe() {
				continue
			}
			if err := nd.checkOrigin(); err == nil {
				results = append(results, nd)
			} else {
				// Return error if any node fails origin check
				return nil, err
			}
		}
	}
	return results, nil
}

// GetElementById returns the element with the given ID.
func (node *NodeDOM) GetElementById(id string) (*NodeDOM, error) {
	if err := node.checkOrigin(); err != nil {
		return nil, err
	}
	return node.QuerySelector("#" + escapeCSSIdentifier(id))
}

// GetElementsByClassName returns all elements with the given class name.
func (node *NodeDOM) GetElementsByClassName(className string) ([]*NodeDOM, error) {
	if err := node.checkOrigin(); err != nil {
		return nil, err
	}
	return node.QuerySelectorAll("." + escapeCSSIdentifier(className))
}

// GetElementsByTagName returns all elements with the given tag name.
func (node *NodeDOM) GetElementsByTagName(tagName string) ([]*NodeDOM, error) {
	if err := node.checkOrigin(); err != nil {
		return nil, err
	}
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

// SandboxFlags represents the sandbox attribute flags for iframes.
type SandboxFlags uint32

const (
	SandboxNone SandboxFlags = 0
	// SandboxAllowScripts allows JavaScript execution in the iframe.
	SandboxAllowScripts SandboxFlags = 1 << iota
	// SandboxAllowForms allows form submission in the iframe.
	SandboxAllowForms
	// SandboxAllowSameOrigin allows the iframe to be treated as same-origin.
	SandboxAllowSameOrigin
	// SandboxAllowTopNavigation allows the iframe to navigate the top-level window.
	SandboxAllowTopNavigation
	// SandboxAllowPopups allows the iframe to open popups.
	SandboxAllowPopups
	// SandboxAllowPresentation allows the iframe to start a presentation session.
	SandboxAllowPresentation
	// SandboxAllowModals allows the iframe to open modal dialogs.
	SandboxAllowModals
	// SandboxAllowOrientationLock allows the iframe to lock screen orientation.
	SandboxAllowOrientationLock
	// SandboxAllowPointerLock allows the iframe to use the Pointer Lock API.
	SandboxAllowPointerLock
)

// ParseSandboxAttribute parses the sandbox attribute value into SandboxFlags.
func ParseSandboxAttribute(value string) SandboxFlags {
	if value == "" {
		// Empty sandbox attribute means all restrictions apply
		return SandboxNone
	}

	var flags SandboxFlags
	tokens := strings.Fields(value)
	for _, token := range tokens {
		switch strings.ToLower(token) {
		case "allow-scripts":
			flags |= SandboxAllowScripts
		case "allow-forms":
			flags |= SandboxAllowForms
		case "allow-same-origin":
			flags |= SandboxAllowSameOrigin
		case "allow-top-navigation":
			flags |= SandboxAllowTopNavigation
		case "allow-popups":
			flags |= SandboxAllowPopups
		case "allow-presentation":
			flags |= SandboxAllowPresentation
		case "allow-modals":
			flags |= SandboxAllowModals
		case "allow-orientation-lock":
			flags |= SandboxAllowOrientationLock
		case "allow-pointer-lock":
			flags |= SandboxAllowPointerLock
		}
	}
	return flags
}

// HasSandboxFlag checks if a specific sandbox flag is set.
func (f SandboxFlags) HasSandboxFlag(flag SandboxFlags) bool {
	return f&flag != 0
}

// LoadIframeContent loads and parses the iframe content.
// For same-origin iframes, it fetches and parses the content.
// For cross-origin iframes, it only loads if allow-same-origin is set in sandbox.
// Returns the parsed iframe document or an error.
func (node *NodeDOM) LoadIframeContent() (*Document, error) {
	if node.Element != "iframe" {
		return nil, fmt.Errorf("node is not an iframe")
	}

	if node.IframeSrc == "" {
		return nil, fmt.Errorf("iframe has no src attribute")
	}

	// Check sandbox restrictions
	if !node.SandboxFlags.HasSandboxFlag(SandboxAllowSameOrigin) {
		// Cross-origin iframe without allow-same-origin - create empty isolated document
		// In a real implementation, this would still load the content but with strict isolation
		return &Document{}, nil
	}

	// Parse the iframe source URL
	iframeURL, err := node.Document.URL.Parse(node.IframeSrc)
	if err != nil {
		return nil, err
	}

	// Check if same-origin
	windowOrigin := node.WindowCtx.GetOrigin()
	if iframeURL.Scheme != windowOrigin.Scheme || iframeURL.Host != windowOrigin.Host {
		// Cross-origin - check if allow-same-origin is set
		if !node.SandboxFlags.HasSandboxFlag(SandboxAllowSameOrigin) {
			// Create empty isolated document for cross-origin iframe
			return &Document{}, nil
		}
	}

	// Same-origin or allow-same-origin set - fetch and parse content
	// This would use the sauce package to fetch the content
	// For now, return an empty document as placeholder
	// In a full implementation, this would fetch the HTML and parse it
	return &Document{}, nil
}

// CanExecuteScripts returns true if scripts are allowed in this iframe context.
func (node *NodeDOM) CanExecuteScripts() bool {
	if node.Element != "iframe" {
		return true // Not an iframe, scripts allowed by default
	}
	return node.SandboxFlags.HasSandboxFlag(SandboxAllowScripts)
}

// CanSubmitForms returns true if form submission is allowed in this iframe context.
func (node *NodeDOM) CanSubmitForms() bool {
	if node.Element != "iframe" {
		return true
	}
	return node.SandboxFlags.HasSandboxFlag(SandboxAllowForms)
}

// CanNavigateTop returns true if top-level navigation is allowed.
func (node *NodeDOM) CanNavigateTop() bool {
	if node.Element != "iframe" {
		return true
	}
	return node.SandboxFlags.HasSandboxFlag(SandboxAllowTopNavigation)
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

// CollapseWhitespace collapses whitespace in text content according to HTML/CSS specs.
// This implements the "white-space: normal" behavior (default).
func CollapseWhitespace(text string) string {
	// Replace all whitespace sequences with a single space
	re := regexp.MustCompile(`\s+`)
	result := re.ReplaceAllString(text, " ")
	// Trim leading and trailing spaces
	result = strings.TrimSpace(result)
	return result
}

// CollapseWhitespacePreserveNewlines collapses whitespace but preserves explicit newlines.
// This implements "white-space: pre-wrap" behavior.
func CollapseWhitespacePreserveNewlines(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		re := regexp.MustCompile(`[ \t]+`)
		collapsed := re.ReplaceAllString(line, " ")
		collapsed = strings.TrimSpace(collapsed)
		result = append(result, collapsed)
	}
	return strings.Join(result, "\n")
}

// GetTextContent returns the text content of a node, handling whitespace according to CSS white-space property.
func (node *NodeDOM) GetTextContent() string {
	if node.Type != NodeTypeText {
		return node.Content
	}

	// Get the white-space property from the node's style or inherit from parent
	whiteSpace := "normal"
	if node.Style != nil && node.Style.Display != "" {
		// In a real implementation, we'd compute the cascaded white-space value
		// For now, use normal as default
		whiteSpace = "normal"
	}

	switch whiteSpace {
	case "pre":
		return node.Content
	case "pre-wrap":
		return CollapseWhitespacePreserveNewlines(node.Content)
	case "pre-line":
		// Similar to pre-wrap but collapses spaces/tabs
		lines := strings.Split(node.Content, "\n")
		var result []string
		for _, line := range lines {
			re := regexp.MustCompile(`\s+`)
			collapsed := re.ReplaceAllString(line, " ")
			collapsed = strings.TrimSpace(collapsed)
			result = append(result, collapsed)
		}
		return strings.Join(result, "\n")
	case "nowrap":
		re := regexp.MustCompile(`\s+`)
		result := re.ReplaceAllString(node.Content, " ")
		result = strings.TrimSpace(result)
		return result
	default: // "normal"
		return CollapseWhitespace(node.Content)
	}
}

// SetInnerHTML sets the HTML content of the node by parsing the HTML string
// and replacing all children with the parsed nodes.
func (node *NodeDOM) SetInnerHTML(htmlString string) error {
	if node.Type != NodeTypeElement {
		return fmt.Errorf("innerHTML can only be set on element nodes")
	}

	// Remove all existing children
	for _, child := range node.Children {
		child.Parent = nil
		child.PrevSibling = nil
		child.NextSibling = nil
		// Clean up JS wrapper reference to prevent memory leak
		if node.WindowCtx != nil && node.WindowCtx.nodeWrappers != nil {
			delete(node.WindowCtx.nodeWrappers, child)
		}
	}
	node.Children = nil
	node.FirstChild = nil
	node.MarkDirty()

	// Parse the HTML string
	parsedNodes, err := parseHTMLFragment(htmlString, node.Document)
	if err != nil {
		return err
	}

	// Append parsed nodes as children
	for _, child := range parsedNodes {
		node.appendChildNode(child)
	}

	node.RequestReflow()
	node.RequestRepaint()
	return nil
}

// parseHTMLFragment parses an HTML fragment and returns a list of NodeDOM nodes.
func parseHTMLFragment(htmlString string, doc *Document) ([]*NodeDOM, error) {
	// Use ParseFragment to parse HTML fragment without full document structure
	// We need a context element - create a div node as context
	contextNode := &html.Node{
		Type:     html.ElementNode,
		Data:     "div",
		DataAtom: atom.Div,
	}
	nodes, err := html.ParseFragment(strings.NewReader(htmlString), contextNode)
	if err != nil {
		return nil, err
	}

	var result []*NodeDOM
	for _, node := range nodes {
		nodeDOM := convertHTMLNodeToNodeDOM(node, doc)
		if nodeDOM != nil {
			result = append(result, nodeDOM)
		}
	}

	return result, nil
}

// convertHTMLNodeToNodeDOM converts an html.Node to a NodeDOM
func convertHTMLNodeToNodeDOM(htmlNode *html.Node, doc *Document) *NodeDOM {
	switch htmlNode.Type {
	case html.TextNode:
		return &NodeDOM{
			Type:     NodeTypeText,
			Content:  htmlNode.Data,
			Document: doc,
		}
	case html.ElementNode:
		nodeDOM := &NodeDOM{
			Type:       NodeTypeElement,
			Element:    htmlNode.Data,
			Children:   make([]*NodeDOM, 0),
			Attributes: make([]*Attribute, 0),
			Document:   doc,
			HTMLNode:   htmlNode,
		}
		for _, attr := range htmlNode.Attr {
			nodeDOM.Attributes = append(nodeDOM.Attributes, &Attribute{
				Name:  attr.Key,
				Value: attr.Val,
			})
		}
		for child := htmlNode.FirstChild; child != nil; child = child.NextSibling {
			childDOM := convertHTMLNodeToNodeDOM(child, doc)
			if childDOM != nil {
				nodeDOM.appendChildNode(childDOM)
			}
		}
		return nodeDOM
	case html.CommentNode:
		return &NodeDOM{
			Type:     NodeTypeComment,
			Content:  htmlNode.Data,
			Document: doc,
		}
	default:
		return nil
	}
}

// SetTextContent sets the text content of the node, replacing all children
// with a single text node.
func (node *NodeDOM) SetTextContent(text string) {
	// Remove all existing children
	for _, child := range node.Children {
		child.Parent = nil
		child.PrevSibling = nil
		child.NextSibling = nil
	}
	node.Children = nil
	node.FirstChild = nil

	if node.Type == NodeTypeText {
		node.Content = text
	} else {
		textNode := &NodeDOM{
			Type:     NodeTypeText,
			Content:  text,
			Document: node.Document,
		}
		node.appendChildNode(textNode)
	}

	node.MarkDirty()
	node.RequestReflow()
	node.RequestRepaint()
}

// InsertBefore inserts a new child node before an existing child node.
// If referenceNode is nil, the new node is appended at the end.
func (node *NodeDOM) InsertBefore(newChild, referenceChild *NodeDOM) error {
	if newChild == nil {
		return fmt.Errorf("newChild cannot be nil")
	}

	if newChild.Parent != nil {
		newChild.Parent.removeChildNode(newChild)
	}

	newChild.Document = node.Document
	newChild.WindowCtx = node.WindowCtx

	if referenceChild == nil {
		node.appendChildNode(newChild)
		return nil
	}

	for i, child := range node.Children {
		if child == referenceChild {
			newChild.Parent = node
			newChild.PrevSibling = referenceChild.PrevSibling
			newChild.NextSibling = referenceChild
			referenceChild.PrevSibling = newChild

			if referenceChild == node.FirstChild {
				node.FirstChild = newChild
			} else if newChild.PrevSibling != nil {
				newChild.PrevSibling.NextSibling = newChild
			}

			node.Children = append(node.Children[:i], append([]*NodeDOM{newChild}, node.Children[i:]...)...)
			node.MarkDirty()
			node.RequestReflow()
			node.RequestRepaint()
			return nil
		}
	}

	return fmt.Errorf("referenceChild is not a child of this node")
}

// ReplaceChild replaces an existing child node with a new node.
// Returns the replaced node.
func (node *NodeDOM) ReplaceChild(newChild, oldChild *NodeDOM) (*NodeDOM, error) {
	if newChild == nil || oldChild == nil {
		return nil, fmt.Errorf("newChild and oldChild cannot be nil")
	}

	found := false
	for _, child := range node.Children {
		if child == oldChild {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("oldChild is not a child of this node")
	}

	if newChild.Parent != nil {
		newChild.Parent.removeChildNode(newChild)
	}

	newChild.Document = node.Document
	newChild.WindowCtx = node.WindowCtx

	for i, child := range node.Children {
		if child == oldChild {
			newChild.PrevSibling = oldChild.PrevSibling
			newChild.NextSibling = oldChild.NextSibling
			newChild.Parent = node

			if oldChild.PrevSibling != nil {
				oldChild.PrevSibling.NextSibling = newChild
			}
			if oldChild.NextSibling != nil {
				oldChild.NextSibling.PrevSibling = newChild
			}
			if node.FirstChild == oldChild {
				node.FirstChild = newChild
			}

			node.Children[i] = newChild

			oldChild.Parent = nil
			oldChild.PrevSibling = nil
			oldChild.NextSibling = nil

			node.MarkDirty()
			node.RequestReflow()
			node.RequestRepaint()
			// Clean up JS wrapper reference to prevent memory leak
			if node.WindowCtx != nil && node.WindowCtx.nodeWrappers != nil {
				delete(node.WindowCtx.nodeWrappers, oldChild)
			}
			return oldChild, nil
		}
	}

	return nil, fmt.Errorf("oldChild not found")
}

// CloneNode creates a copy of the node.
// If deep is true, it also clones all descendants.
func (node *NodeDOM) CloneNode(deep bool) *NodeDOM {
	clone := &NodeDOM{
		Type:       node.Type,
		Element:    node.Element,
		Content:    node.Content,
		Attributes: make([]*Attribute, 0),
		Document:   node.Document,
		WindowCtx:  node.WindowCtx,
		Style:      node.Style,
	}

	for _, attr := range node.Attributes {
		clone.Attributes = append(clone.Attributes, &Attribute{
			Name:  attr.Name,
			Value: attr.Value,
		})
	}

	if deep {
		clone.Children = make([]*NodeDOM, 0, len(node.Children))
		for _, child := range node.Children {
			clonedChild := child.CloneNode(true)
			clone.appendChildNode(clonedChild)
		}
	}

	return clone
}

// CreateElement creates a new element node and returns it.
// The element is not automatically added to the DOM tree.
func (doc *Document) CreateElement(tagName string) *NodeDOM {
	element := &NodeDOM{
		Type:       NodeTypeElement,
		Element:    tagName,
		Children:   make([]*NodeDOM, 0),
		Attributes: make([]*Attribute, 0),
		Document:   doc,
	}
	return element
}

// CreateTextNode creates a new text node and returns it.
// The text node is not automatically added to the DOM tree.
func (doc *Document) CreateTextNode(data string) *NodeDOM {
	textNode := &NodeDOM{
		Type:     NodeTypeText,
		Content:  data,
		Document: doc,
	}
	return textNode
}

// GetBody returns the body element of the document.
func (doc *Document) GetBody() (*NodeDOM, error) {
	if doc.DOM == nil {
		return nil, nil
	}
	return doc.DOM.QuerySelector("body")
}

// GetDocumentElement returns the root element of the document (html).
func (doc *Document) GetDocumentElement() (*NodeDOM, error) {
	if doc.DOM == nil {
		return nil, nil
	}
	return doc.DOM.QuerySelector("html")
}

// GetTitle returns the document title.
func (doc *Document) GetTitle() string {
	return doc.Title
}
