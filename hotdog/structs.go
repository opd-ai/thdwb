package hotdog

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	cascadia "github.com/andybalholm/cascadia"
	"github.com/danfragoso/thdwb/assets"
	"github.com/danfragoso/thdwb/mustard"
	profiler "github.com/danfragoso/thdwb/profiler"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type WebBrowser struct {
	ActiveDocument *Document
	Documents      []*Document

	Viewport    *mustard.CanvasWidget
	StatusLabel *mustard.LabelWidget
	History     *History
	Window      *mustard.Window
	Profiler    *profiler.Profiler
	BuildInfo   *assets.BuildInfo
	Settings    *Settings
}

type Document struct {
	Title       string
	ContentType string
	URL         *url.URL

	RawDocument string
	DOM         *NodeDOM
	HTMLRoot    *html.Node // Root of the parsed HTML tree for CSS selector queries

	StyleSheets []*StyleElement // Parsed CSS stylesheets from <style> elements

	// Content Security Policy
	CSP             *ContentSecurityPolicy
	ResponseHeaders http.Header

	DebugFlag       bool
	DebugWindow     *mustard.Window
	DebugTree       *mustard.TreeWidget
	SelectedElement *NodeDOM

	OffsetY int
}

type History struct {
	previousPages []*url.URL
	nextPages     []*url.URL
}

func (history *History) NextPages() []*url.URL {
	return history.nextPages
}

func (history *History) AllPages() []*url.URL {
	return history.previousPages
}

func (history *History) PageCount() int {
	return len(history.previousPages)
}

func (history *History) Push(URL *url.URL) {
	history.nextPages = nil
	history.previousPages = append(history.previousPages, URL)
}

func (history *History) Last() *url.URL {
	if len(history.previousPages) == 0 {
		return nil
	}
	return history.previousPages[len(history.previousPages)-1]
}

func (history *History) PopNext() {
	if len(history.nextPages) > 0 {
		history.previousPages = append(history.previousPages, history.nextPages[len(history.nextPages)-1])
		history.nextPages = nil
	}
}

func (history *History) Pop() {
	if len(history.previousPages) > 0 {
		history.nextPages = append(history.nextPages, history.previousPages[len(history.previousPages)-1])
		history.previousPages = history.previousPages[:len(history.previousPages)-1]
	}
}

type RenderBox struct {
	Node *NodeDOM

	Top  float64
	Left float64

	Width  float64
	Height float64

	MarginTop    float64
	MarginLeft   float64
	MarginRight  float64
	MarginBottom float64

	PaddingTop    float64
	PaddingLeft   float64
	PaddingRight  float64
	PaddingBottom float64
}

func (box *RenderBox) GetRect() (float64, float64, float64, float64) {
	return box.Top, box.Left, box.Width, box.Height
}

type NoSuchElementError string

func (e NoSuchElementError) Error() string {
	return fmt.Sprintf("no such element: %q", string(e))
}

// Resource "HTTP resource struct definition"
type Resource struct {
	Body        string
	ContentType string
	Code        int
	URL         *url.URL
	Key         string
	Headers     http.Header
}

// Attribute "Generic key:value attribute definition"
type Attribute struct {
	Name  string
	Value string
}

// Stylesheet "Stylesheet definition for DOM Nodes"
type Stylesheet struct {
	Color           *ColorRGBA
	BackgroundColor *ColorRGBA

	FontSize       float64
	FontWeight     int
	FontFamily     string
	FontStyle      string
	LineHeight     float64
	LetterSpacing  float64
	TextAlign      string
	TextDecoration string
	WhiteSpace     string

	Display        string
	Position       string
	FlexDirection  string
	FlexWrap       string
	JustifyContent string
	AlignItems     string
	AlignContent   string
	FlexGrow       float64
	FlexShrink     float64
	FlexBasis      string
	Order          int
	AlignSelf      string

	Width     float64
	Height    float64
	MinWidth  float64
	MinHeight float64
	MaxWidth  float64
	MaxHeight float64
	Top       float64
	Right     float64
	Bottom    float64
	Left      float64
	ZIndex    int

	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64

	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64

	BorderTopWidth    float64
	BorderRightWidth  float64
	BorderBottomWidth float64
	BorderLeftWidth   float64
	BorderTopStyle    string
	BorderRightStyle  string
	BorderBottomStyle string
	BorderLeftStyle   string
	BorderTopColor    *ColorRGBA
	BorderRightColor  *ColorRGBA
	BorderBottomColor *ColorRGBA
	BorderLeftColor   *ColorRGBA

	OverflowX  string
	OverflowY  string
	Visibility string
	Opacity    float64
}

// StyleElement "hmtl <style> element"
type StyleElement struct {
	Selector    string
	Style       *Stylesheet
	Specificity cascadia.Specificity // CSS selector specificity for cascade resolution
	Origin      string               // Origin: "user-agent", "user", "author"
	Index       int                  // Order index for stable sorting
}

// ColorRGBA "RGBA color model"
type ColorRGBA struct {
	R float64
	G float64
	B float64
	A float64
}

type ResourceCache struct {
	cachedResources []*Resource
}

func (cache *ResourceCache) AddResource(resource *Resource) {
	cache.cachedResources = append(cache.cachedResources, resource)
}

func (cache *ResourceCache) GetResource(resourceKey string) *Resource {
	for _, resource := range cache.cachedResources {
		if resource.Key == resourceKey {
			return resource
		}
	}

	return nil
}

type CachedImage struct {
	Key   string
	Image []byte
}
type ImgCache struct {
	cachedImages []*CachedImage
}

func (cache *ImgCache) AddImage(key string, value []byte) {
	cache.cachedImages = append(cache.cachedImages,
		&CachedImage{
			Key:   key,
			Image: value,
		},
	)
}

func (cache *ImgCache) GetImage(imageKey string) *CachedImage {
	for _, image := range cache.cachedImages {
		if image.Key == imageKey {
			return image
		}
	}

	return nil
}

func Log(component, msg string) {
	str := "(" + "\033[95m" + component + "\033[0m" + ")"
	fmt.Println(str, msg)
}

// ContentSecurityPolicy represents a parsed Content Security Policy.
// It enforces restrictions on resources, scripts, styles, etc.
type ContentSecurityPolicy struct {
	// Directive name -> list of allowed sources
	directives map[string][]string
	origin     *url.URL
	additional []*ContentSecurityPolicy
	// Report URI for violations
	ReportURI string
	// Whether this is a report-only policy
	ReportOnly bool
}

// CSPDirective represents a CSP directive with its allowed sources.
type CSPDirective struct {
	Name    string
	Sources []string
}

// NewContentSecurityPolicy creates a new CSP from a policy string (e.g., from HTTP header).
func NewContentSecurityPolicy(policy string) *ContentSecurityPolicy {
	csp := &ContentSecurityPolicy{
		directives: make(map[string][]string),
	}
	csp.parse(policy)
	return csp
}

// ApplyContentSecurityPolicyHeaders parses an enforcing CSP response header.
func (document *Document) ApplyContentSecurityPolicyHeaders(headers http.Header) {
	document.ResponseHeaders = headers.Clone()
	policies := headers.Values("Content-Security-Policy")
	if len(policies) == 0 {
		document.CSP = nil
		return
	}
	document.CSP = NewContentSecurityPolicy(policies[0])
	document.CSP.origin = document.URL
	for _, policy := range policies[1:] {
		additional := NewContentSecurityPolicy(policy)
		additional.origin = document.URL
		document.CSP.additional = append(document.CSP.additional, additional)
	}
}

// parse parses a CSP policy string into directives.
func (csp *ContentSecurityPolicy) parse(policy string) {
	if policy == "" {
		return
	}
	directives := strings.Split(policy, ";")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}
		parts := strings.Fields(directive)
		if len(parts) == 0 {
			continue
		}
		name := strings.ToLower(parts[0])
		if _, exists := csp.directives[name]; exists {
			continue
		}
		sources := parts[1:]
		csp.directives[name] = sources
	}
}

// AllowInlineScript checks if inline scripts are allowed by CSP.
// This is used for eval(), Function(), and inline event handlers.
func (csp *ContentSecurityPolicy) AllowInlineScript() bool {
	if csp == nil {
		return true // No CSP = allow all
	}
	if !csp.allowInlineScriptPolicy() {
		return false
	}
	for _, additional := range csp.additional {
		if !additional.AllowInlineScript() {
			return false
		}
	}
	return true
}

func (csp *ContentSecurityPolicy) allowInlineScriptPolicy() bool {
	sources := csp.directives["script-src"]
	if sources == nil {
		sources = csp.directives["default-src"]
	}
	if sources == nil {
		return true // No script-src = allow all
	}
	for _, src := range sources {
		if strings.EqualFold(src, "'unsafe-inline'") {
			return true
		}
	}
	return false
}

// AllowEval checks if eval() and Function() are allowed by CSP.
func (csp *ContentSecurityPolicy) AllowEval() bool {
	if csp == nil {
		return true
	}
	if !csp.allowEvalPolicy() {
		return false
	}
	for _, additional := range csp.additional {
		if !additional.AllowEval() {
			return false
		}
	}
	return true
}

func (csp *ContentSecurityPolicy) allowEvalPolicy() bool {
	sources := csp.directives["script-src"]
	if sources == nil {
		sources = csp.directives["default-src"]
	}
	if sources == nil {
		return true
	}
	for _, src := range sources {
		if strings.EqualFold(src, "'unsafe-eval'") {
			return true
		}
	}
	return false
}

// AllowInlineStyle checks if inline styles are allowed by CSP.
func (csp *ContentSecurityPolicy) AllowInlineStyle() bool {
	if csp == nil {
		return true
	}
	if !csp.allowInlineStylePolicy() {
		return false
	}
	for _, additional := range csp.additional {
		if !additional.AllowInlineStyle() {
			return false
		}
	}
	return true
}

func (csp *ContentSecurityPolicy) allowInlineStylePolicy() bool {
	sources := csp.directives["style-src"]
	if sources == nil {
		sources = csp.directives["default-src"]
	}
	if sources == nil {
		return true
	}
	for _, src := range sources {
		if strings.EqualFold(src, "'unsafe-inline'") {
			return true
		}
	}
	return false
}

// CheckStyleSource checks if a style source is allowed by CSP.
// Used for style attribute and <style> elements.
func (csp *ContentSecurityPolicy) CheckStyleSource(source string) bool {
	return csp.checkSource("style-src", source)
}

// CheckScriptSource checks if a script source is allowed by CSP.
// Used for script src attributes and dynamic script creation.
func (csp *ContentSecurityPolicy) CheckScriptSource(source string) bool {
	return csp.checkSource("script-src", source)
}

// CheckImageSource checks whether an image source is permitted by img-src.
func (csp *ContentSecurityPolicy) CheckImageSource(source string) bool {
	return csp.checkSource("img-src", source)
}

func (csp *ContentSecurityPolicy) checkSource(directive, source string) bool {
	if csp == nil {
		return true
	}
	if csp.origin != nil {
		parsed, err := url.Parse(source)
		if err == nil && !parsed.IsAbs() {
			source = csp.origin.ResolveReference(parsed).String()
		}
	}
	if !csp.checkSourcePolicy(directive, source) {
		return false
	}
	for _, additional := range csp.additional {
		if !additional.checkSource(directive, source) {
			return false
		}
	}
	return true
}

func (csp *ContentSecurityPolicy) checkSourcePolicy(directive, source string) bool {
	sources := csp.directives[directive]
	if sources == nil {
		sources = csp.directives["default-src"]
	}
	if sources == nil {
		return true
	}
	if len(sources) == 1 && strings.EqualFold(sources[0], "'none'") {
		return false
	}
	for _, src := range sources {
		switch {
		case strings.EqualFold(src, "'self'") && csp.matchesOrigin(source):
			return true
		case src == "*" && allowsWildcardSource(source):
			return true
		case strings.HasSuffix(src, ":") && strings.EqualFold(src, schemeOf(source)):
			return true
		case matchesSourceURL(src, source):
			return true
		}
	}
	return false
}

func allowsWildcardSource(source string) bool {
	switch schemeOf(source) {
	case "http:", "https:", "ws:", "wss:":
		return true
	default:
		return false
	}
}

func matchesSourceURL(allowed, source string) bool {
	allowedURL, allowedErr := url.Parse(allowed)
	sourceURL, sourceErr := url.Parse(source)
	if allowedErr != nil || sourceErr != nil || allowedURL.Host == "" || sourceURL.Host == "" {
		return strings.EqualFold(allowed, source)
	}
	if originKey(allowedURL) != originKey(sourceURL) {
		return false
	}
	allowedPath := allowedURL.EscapedPath()
	sourcePath := sourceURL.EscapedPath()
	if allowedPath == "" || allowedPath == "/" {
		return true
	}
	if strings.HasSuffix(allowedPath, "/") {
		return strings.HasPrefix(sourcePath, allowedPath)
	}
	return allowedPath == sourcePath
}

func (csp *ContentSecurityPolicy) matchesOrigin(source string) bool {
	if csp.origin == nil {
		return strings.EqualFold(source, "self")
	}

	parsed, err := url.Parse(source)
	if err != nil {
		return false
	}
	return originKey(parsed) == originKey(csp.origin)
}

func schemeOf(source string) string {
	parsed, err := url.Parse(source)
	if err != nil {
		return ""
	}
	return parsed.Scheme + ":"
}

// CheckMarkup reports the first inline script, style, event handler, or resource denied by CSP.
func (csp *ContentSecurityPolicy) CheckMarkup(markup string) (string, string) {
	if csp == nil {
		return "", ""
	}
	contextNode := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(markup), contextNode)
	if err != nil {
		return "", ""
	}
	for _, node := range nodes {
		if directive, blocked := csp.checkMarkupNode(node); directive != "" {
			return directive, blocked
		}
	}
	return "", ""
}

// ValidateMarkup applies this document's CSP to inline and resource markup.
func (document *Document) ValidateMarkup(markup string) error {
	if document == nil || document.CSP == nil {
		return nil
	}
	directive, blocked := document.CSP.CheckMarkup(markup)
	if directive == "" {
		return nil
	}
	return document.reportCSPViolation(directive, blocked)
}

// CheckCSPResource validates a script, style, or image URL against the document policy.
func (document *Document) CheckCSPResource(kind, source string) error {
	if kind != "script" && kind != "style" && kind != "img" {
		return fmt.Errorf("unsupported CSP resource type %q", kind)
	}
	if document == nil || document.CSP == nil {
		return nil
	}
	directive := kind + "-src"
	allowed := document.CSP.checkSource(directive, source)
	if allowed {
		return nil
	}
	return document.reportCSPViolation(directive, source)
}

func (document *Document) reportCSPViolation(directive, blocked string) error {
	origin := "unknown"
	if document.URL != nil {
		origin = document.URL.String()
	}
	message := fmt.Sprintf("Content Security Policy blocked %q by %s", blocked, directive)
	Log(fmt.Sprintf("[console.error] [origin: %s]", origin), message)
	return SecurityError(message)
}

func (csp *ContentSecurityPolicy) checkMarkupNode(node *html.Node) (string, string) {
	if node.Type == html.ElementNode {
		if directive, blocked := csp.checkMarkupElement(node); directive != "" {
			return directive, blocked
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if directive, blocked := csp.checkMarkupNode(child); directive != "" {
			return directive, blocked
		}
	}
	return "", ""
}

func (csp *ContentSecurityPolicy) checkMarkupElement(node *html.Node) (string, string) {
	if node.Data == "script" {
		if source := htmlAttribute(node, "src"); source != "" {
			if !csp.CheckScriptSource(source) {
				return "script-src", source
			}
		} else if !csp.AllowInlineScript() {
			return "script-src", "inline script"
		}
	}
	if node.Data == "style" && !csp.AllowInlineStyle() {
		return "style-src", "inline style"
	}
	for _, attr := range node.Attr {
		if strings.HasPrefix(strings.ToLower(attr.Key), "on") && !csp.AllowInlineScript() {
			return "script-src", "inline event handler"
		}
		if strings.EqualFold(attr.Key, "style") && !csp.AllowInlineStyle() {
			return "style-src", "style attribute"
		}
		if node.Data == "img" && strings.EqualFold(attr.Key, "src") && !csp.CheckImageSource(attr.Val) {
			return "img-src", attr.Val
		}
		if node.Data == "link" && strings.EqualFold(attr.Key, "href") &&
			strings.Contains(strings.ToLower(htmlAttribute(node, "rel")), "stylesheet") &&
			!csp.CheckStyleSource(attr.Val) {
			return "style-src", attr.Val
		}
	}
	return "", ""
}

func htmlAttribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}
