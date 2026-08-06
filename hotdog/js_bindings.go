package hotdog

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

// nodeWrapperSymbol is a unique symbol used to store the JSDOMWrapper reference on JS objects.
var nodeWrapperSymbol = goja.NewSymbol("thdwb.nodeWrapper")

// JSDOMWrapper wraps a NodeDOM for JavaScript access
type JSDOMWrapper struct {
	node      *NodeDOM
	runtime   *goja.Runtime
	windowCtx *WindowContext
}

// JSDocumentWrapper wraps a Document for JavaScript access
type JSDocumentWrapper struct {
	doc       *Document
	runtime   *goja.Runtime
	windowCtx *WindowContext
}

// JSWindowWrapper wraps a WindowContext for JavaScript access
type JSWindowWrapper struct {
	windowCtx *WindowContext
	runtime   *goja.Runtime
}

// JSConsoleWrapper provides console.log, console.error, console.warn
type JSConsoleWrapper struct {
	windowCtx *WindowContext
	runtime   *goja.Runtime
}

// InitJSRuntime initializes the JavaScript runtime with DOM bindings
func (wc *WindowContext) InitJSRuntime() error {
	runtime := wc.GetJSRuntime()
	if runtime == nil {
		return fmt.Errorf("no JavaScript runtime available")
	}

	// Create console object
	console := &JSConsoleWrapper{
		windowCtx: wc,
		runtime:   runtime,
	}
	consoleObj := runtime.NewObject()
	consoleObj.Set("log", console.log)
	consoleObj.Set("error", console.error)
	consoleObj.Set("warn", console.warn)
	runtime.Set("console", consoleObj)

	// Create document object
	if wc.ActiveDocument != nil {
		docWrapper := &JSDocumentWrapper{
			doc:       wc.ActiveDocument,
			runtime:   runtime,
			windowCtx: wc,
		}
		docObj := runtime.NewObject()
		docObj.Set("querySelector", docWrapper.querySelector)
		docObj.Set("querySelectorAll", docWrapper.querySelectorAll)
		docObj.Set("getElementById", docWrapper.getElementById)
		docObj.Set("getElementsByClassName", docWrapper.getElementsByClassName)
		docObj.Set("getElementsByTagName", docWrapper.getElementsByTagName)
		docObj.Set("createElement", docWrapper.createElement)
		docObj.Set("createTextNode", docWrapper.createTextNode)
		// Expose body and documentElement as properties (getters)
		docObj.DefineAccessorProperty("body", runtime.ToValue(docWrapper.getBody), goja.Undefined(), 0, 0)
		docObj.DefineAccessorProperty("documentElement", runtime.ToValue(docWrapper.getDocumentElement), goja.Undefined(), 0, 0)
		docObj.Set("title", docWrapper.getTitle)
		runtime.Set("document", docObj)
	}

	// Create window object
	windowWrapper := &JSWindowWrapper{
		windowCtx: wc,
		runtime:   runtime,
	}
	windowObj := runtime.NewObject()
	windowObj.Set("document", runtime.Get("document"))
	windowObj.Set("console", runtime.Get("console"))
	windowObj.Set("location", windowWrapper.getLocation)
	windowObj.Set("localStorage", windowWrapper.getLocalStorage)
	windowObj.Set("sessionStorage", windowWrapper.getSessionStorage)
	windowObj.Set("addEventListener", windowWrapper.addEventListener)
	windowObj.Set("removeEventListener", windowWrapper.removeEventListener)
	windowObj.Set("postMessage", windowWrapper.postMessage)
	windowObj.Set("fetch", windowWrapper.fetch)
	windowObj.Set("XMLHttpRequest", windowWrapper.XMLHttpRequest)
	runtime.Set("window", windowObj)
	runtime.Set("self", windowObj) // window.self === window

	// Block dangerous APIs unless CSP allows
	runtime.Set("eval", func(call goja.FunctionCall) goja.Value {
		// Check CSP for 'unsafe-eval' from the active document
		if wc.ActiveDocument != nil && wc.ActiveDocument.CSP != nil {
			if wc.ActiveDocument.CSP.AllowEval() {
				// CSP allows eval, but we still don't implement it for security
				// In a real browser, this would execute the code
				return goja.Undefined()
			}
		}
		panic(runtime.NewGoError(SecurityError("eval() is blocked by CSP")))
	})
	runtime.Set("Function", func(call goja.FunctionCall) goja.Value {
		// Check CSP for 'unsafe-eval' from the active document
		if wc.ActiveDocument != nil && wc.ActiveDocument.CSP != nil {
			if wc.ActiveDocument.CSP.AllowEval() {
				// CSP allows Function constructor, but we still don't implement it for security
				return goja.Undefined()
			}
		}
		panic(runtime.NewGoError(SecurityError("Function constructor is blocked by CSP")))
	})

	return nil
}

// === JSConsoleWrapper methods ===

func (c *JSConsoleWrapper) log(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	Log(fmt.Sprintf("[console.log] [origin: %s]", c.windowCtx.GetOrigin().String()), fmt.Sprint(args...))
	return goja.Undefined()
}

func (c *JSConsoleWrapper) error(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	Log(fmt.Sprintf("[console.error] [origin: %s]", c.windowCtx.GetOrigin().String()), fmt.Sprint(args...))
	return goja.Undefined()
}

func (c *JSConsoleWrapper) warn(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	Log(fmt.Sprintf("[console.warn] [origin: %s]", c.windowCtx.GetOrigin().String()), fmt.Sprint(args...))
	return goja.Undefined()
}

// === JSDocumentWrapper methods ===

// querySelector returns the first element matching the CSS selector.
func (d *JSDocumentWrapper) querySelector(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("querySelector requires a selector argument")))
	}
	selector := call.Arguments[0].String()
	if d.doc.DOM == nil {
		return goja.Null()
	}
	// Check origin on the document's root node
	if err := d.doc.DOM.checkOrigin(); err != nil {
		panic(d.runtime.NewGoError(err))
	}
	result, err := d.doc.DOM.QuerySelector(selector)
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	if result == nil {
		return goja.Null()
	}
	return d.wrapNode(result)
}

// querySelectorAll returns all elements matching the CSS selector.
func (d *JSDocumentWrapper) querySelectorAll(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("querySelectorAll requires a selector argument")))
	}
	selector := call.Arguments[0].String()
	if d.doc.DOM == nil {
		return d.runtime.NewArray()
	}
	// Check origin on the document's root node
	if err := d.doc.DOM.checkOrigin(); err != nil {
		panic(d.runtime.NewGoError(err))
	}
	results, err := d.doc.DOM.QuerySelectorAll(selector)
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	arr := d.runtime.NewArray()
	for i, node := range results {
		arr.Set(strconv.Itoa(i), d.wrapNode(node))
	}
	return arr
}

// getElementById returns the element with the given ID.
func (d *JSDocumentWrapper) getElementById(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("getElementById requires an id argument")))
	}
	id := call.Arguments[0].String()
	if d.doc.DOM == nil {
		return goja.Null()
	}
	// Check origin on the document's root node
	if err := d.doc.DOM.checkOrigin(); err != nil {
		panic(d.runtime.NewGoError(err))
	}
	result, err := d.doc.DOM.GetElementById(id)
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	if result == nil {
		return goja.Null()
	}
	return d.wrapNode(result)
}

// getElementsByClassName returns all elements with the given class name.
func (d *JSDocumentWrapper) getElementsByClassName(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("getElementsByClassName requires a class name argument")))
	}
	className := call.Arguments[0].String()
	if d.doc.DOM == nil {
		return d.runtime.NewArray()
	}
	// Check origin on the document's root node
	if err := d.doc.DOM.checkOrigin(); err != nil {
		panic(d.runtime.NewGoError(err))
	}
	results, err := d.doc.DOM.GetElementsByClassName(className)
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	arr := d.runtime.NewArray()
	for i, node := range results {
		arr.Set(strconv.Itoa(i), d.wrapNode(node))
	}
	return arr
}

// getElementsByTagName returns all elements with the given tag name.
func (d *JSDocumentWrapper) getElementsByTagName(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("getElementsByTagName requires a tag name argument")))
	}
	tagName := call.Arguments[0].String()
	if d.doc.DOM == nil {
		return d.runtime.NewArray()
	}
	// Check origin on the document's root node
	if err := d.doc.DOM.checkOrigin(); err != nil {
		panic(d.runtime.NewGoError(err))
	}
	results, err := d.doc.DOM.GetElementsByTagName(tagName)
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	arr := d.runtime.NewArray()
	for i, node := range results {
		arr.Set(strconv.Itoa(i), d.wrapNode(node))
	}
	return arr
}

// createElement creates a new element node.
func (d *JSDocumentWrapper) createElement(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("createElement requires a tag name argument")))
	}
	tagName := call.Arguments[0].String()
	element := d.doc.CreateElement(tagName)
	element.WindowCtx = d.windowCtx
	return d.wrapNode(element)
}

// createTextNode creates a new text node.
func (d *JSDocumentWrapper) createTextNode(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.runtime.NewGoError(errors.New("createTextNode requires a data argument")))
	}
	data := call.Arguments[0].String()
	textNode := d.doc.CreateTextNode(data)
	textNode.WindowCtx = d.windowCtx
	return d.wrapNode(textNode)
}

// getBody returns the body element.
func (d *JSDocumentWrapper) getBody(call goja.FunctionCall) goja.Value {
	// Check origin on the document's root node
	if d.doc.DOM != nil {
		if err := d.doc.DOM.checkOrigin(); err != nil {
			panic(d.runtime.NewGoError(err))
		}
	}
	body, err := d.doc.GetBody()
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	if body == nil {
		return goja.Null()
	}
	return d.wrapNode(body)
}

// getDocumentElement returns the document element (html).
func (d *JSDocumentWrapper) getDocumentElement(call goja.FunctionCall) goja.Value {
	// Check origin on the document's root node
	if d.doc.DOM != nil {
		if err := d.doc.DOM.checkOrigin(); err != nil {
			panic(d.runtime.NewGoError(err))
		}
	}
	docEl, err := d.doc.GetDocumentElement()
	if err != nil {
		panic(d.runtime.NewGoError(err))
	}
	if docEl == nil {
		return goja.Null()
	}
	return d.wrapNode(docEl)
}

// getTitle returns the document title.
func (d *JSDocumentWrapper) getTitle(call goja.FunctionCall) goja.Value {
	return d.runtime.ToValue(d.doc.GetTitle())
}

// wrapNode wraps a NodeDOM in a JSDOMWrapper and returns a JS object.
func (d *JSDocumentWrapper) wrapNode(node *NodeDOM) goja.Value {
	wrapper := &JSDOMWrapper{
		node:      node,
		runtime:   d.runtime,
		windowCtx: d.windowCtx,
	}
	obj := d.runtime.NewObject()
	obj.SetSymbol(nodeWrapperSymbol, wrapper) // Store wrapper reference for retrieval
	obj.Set("nodeType", node.Type)
	obj.Set("tagName", strings.ToUpper(node.Element))
	// Define textContent with getter/setter
	obj.DefineAccessorProperty("textContent", d.runtime.ToValue(wrapper.getTextContentProp), d.runtime.ToValue(wrapper.setTextContent), 0, 0)
	// Define innerHTML with getter/setter
	obj.DefineAccessorProperty("innerHTML", d.runtime.ToValue(wrapper.getInnerHTMLProp), d.runtime.ToValue(wrapper.setInnerHTML), 0, 0)
	obj.Set("id", node.Attr("id"))
	obj.Set("className", node.Attr("class"))
	obj.Set("querySelector", wrapper.querySelector)
	obj.Set("querySelectorAll", wrapper.querySelectorAll)
	obj.Set("getAttribute", wrapper.getAttribute)
	obj.Set("setAttribute", wrapper.setAttribute)
	obj.Set("removeAttribute", wrapper.removeAttribute)
	obj.Set("appendChild", wrapper.appendChild)
	obj.Set("removeChild", wrapper.removeChild)
	obj.Set("insertBefore", wrapper.insertBefore)
	obj.Set("replaceChild", wrapper.replaceChild)
	obj.Set("cloneNode", wrapper.cloneNode)
	obj.Set("parentNode", wrapper.getParentNode)
	obj.Set("childNodes", wrapper.getChildNodes)
	obj.Set("firstChild", wrapper.getFirstChild)
	obj.Set("lastChild", wrapper.getLastChild)
	obj.Set("nextSibling", wrapper.getNextSibling)
	obj.Set("previousSibling", wrapper.getPreviousSibling)
	obj.Set("style", wrapper.getStyle)
	return obj
}

// JSDOMWrapper methods for DOM manipulation

// querySelector returns the first element matching the selector within this node's subtree.
func (w *JSDOMWrapper) querySelector(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("querySelector requires a selector argument")))
	}
	selector := call.Arguments[0].String()
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	result, err := w.node.QuerySelector(selector)
	if err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if result == nil {
		return goja.Null()
	}
	return w.wrapNode(result)
}

// querySelectorAll returns all elements matching the selector within this node's subtree.
func (w *JSDOMWrapper) querySelectorAll(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("querySelectorAll requires a selector argument")))
	}
	selector := call.Arguments[0].String()
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	results, err := w.node.QuerySelectorAll(selector)
	if err != nil {
		panic(w.runtime.NewGoError(err))
	}
	arr := w.runtime.NewArray()
	for i, node := range results {
		arr.Set(strconv.Itoa(i), w.wrapNode(node))
	}
	return arr
}

// getAttribute returns the value of the specified attribute.
func (w *JSDOMWrapper) getAttribute(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("getAttribute requires an attribute name argument")))
	}
	name := call.Arguments[0].String()
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	value := w.node.Attr(name)
	if value == "" {
		return goja.Null()
	}
	return w.runtime.ToValue(value)
}

// setAttribute sets the value of the specified attribute.
func (w *JSDOMWrapper) setAttribute(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(w.runtime.NewGoError(errors.New("setAttribute requires name and value arguments")))
	}
	name := call.Arguments[0].String()
	value := call.Arguments[1].String()
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}

	// Check CSP for event handler attributes and style attribute
	if w.node.Document != nil && w.node.Document.CSP != nil {
		csp := w.node.Document.CSP
		lowerName := strings.ToLower(name)
		// Check for event handler attributes (onclick, onload, etc.)
		if strings.HasPrefix(lowerName, "on") && !csp.AllowInlineScript() {
			panic(w.runtime.NewGoError(SecurityError("event handler attribute blocked by CSP: inline scripts not allowed")))
		}
		// Check for style attribute
		if lowerName == "style" && !csp.AllowInlineStyle() {
			panic(w.runtime.NewGoError(SecurityError("style attribute blocked by CSP: inline styles not allowed")))
		}
	}

	found := false
	for _, attr := range w.node.Attributes {
		if attr.Name == name {
			attr.Value = value
			found = true
			break
		}
	}
	if !found {
		w.node.Attributes = append(w.node.Attributes, &Attribute{Name: name, Value: value})
	}
	w.node.RequestReflow()
	w.node.RequestRepaint()
	return goja.Undefined()
}

// removeAttribute removes the specified attribute.
func (w *JSDOMWrapper) removeAttribute(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("removeAttribute requires an attribute name argument")))
	}
	name := call.Arguments[0].String()
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	newAttrs := make([]*Attribute, 0, len(w.node.Attributes))
	for _, attr := range w.node.Attributes {
		if attr.Name != name {
			newAttrs = append(newAttrs, attr)
		}
	}
	w.node.Attributes = newAttrs
	w.node.RequestReflow()
	w.node.RequestRepaint()
	return goja.Undefined()
}

// appendChild appends a child node to this node.
func (w *JSDOMWrapper) appendChild(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("appendChild requires a node argument")))
	}
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	childVal := call.Arguments[0]
	childObj := childVal.ToObject(w.runtime)
	if childObj == nil {
		panic(w.runtime.NewGoError(errors.New("appendChild argument must be a Node object")))
	}
	childWrapper := w.getNodeFromJSObject(childObj)
	if childWrapper == nil {
		panic(w.runtime.NewGoError(errors.New("appendChild argument must be a valid Node")))
	}
	childNode := childWrapper.node
	if err := childNode.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if childNode.Parent != nil {
		childNode.Parent.removeChildNode(childNode)
	}
	w.node.appendChildNode(childNode)
	w.node.RequestReflow()
	w.node.RequestRepaint()
	return w.wrapNode(childNode)
}

// removeChild removes a child node from this node.
func (w *JSDOMWrapper) removeChild(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("removeChild requires a node argument")))
	}
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	childVal := call.Arguments[0]
	childObj := childVal.ToObject(w.runtime)
	if childObj == nil {
		panic(w.runtime.NewGoError(errors.New("removeChild argument must be a Node object")))
	}
	childWrapper := w.getNodeFromJSObject(childObj)
	if childWrapper == nil {
		panic(w.runtime.NewGoError(errors.New("removeChild argument must be a valid Node")))
	}
	childNode := childWrapper.node
	if err := childNode.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	w.node.removeChildNode(childNode)
	w.node.RequestReflow()
	w.node.RequestRepaint()
	return w.wrapNode(childNode)
}

// getParentNode returns the parent node.
func (w *JSDOMWrapper) getParentNode(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if w.node.Parent == nil {
		return goja.Null()
	}
	return w.wrapNode(w.node.Parent)
}

// getChildNodes returns a NodeList of child nodes.
func (w *JSDOMWrapper) getChildNodes(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	arr := w.runtime.NewArray()
	for i, child := range w.node.Children {
		arr.Set(strconv.Itoa(i), w.wrapNode(child))
	}
	return arr
}

// getFirstChild returns the first child node.
func (w *JSDOMWrapper) getFirstChild(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if len(w.node.Children) == 0 {
		return goja.Null()
	}
	return w.wrapNode(w.node.Children[0])
}

// getLastChild returns the last child node.
func (w *JSDOMWrapper) getLastChild(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if len(w.node.Children) == 0 {
		return goja.Null()
	}
	return w.wrapNode(w.node.Children[len(w.node.Children)-1])
}

// getNextSibling returns the next sibling node.
func (w *JSDOMWrapper) getNextSibling(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if w.node.NextSibling == nil {
		return goja.Null()
	}
	return w.wrapNode(w.node.NextSibling)
}

// getPreviousSibling returns the previous sibling node.
func (w *JSDOMWrapper) getPreviousSibling(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if w.node.PrevSibling == nil {
		return goja.Null()
	}
	return w.wrapNode(w.node.PrevSibling)
}

// setInnerHTML sets the HTML content of the element.
func (w *JSDOMWrapper) setInnerHTML(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("innerHTML setter requires a value argument")))
	}
	htmlString := call.Arguments[0].String()

	// Check CSP for inline scripts and styles
	if w.node.Document != nil && w.node.Document.CSP != nil {
		csp := w.node.Document.CSP
		// Check if the HTML contains script tags or event handlers
		if strings.Contains(strings.ToLower(htmlString), "<script") && !csp.AllowInlineScript() {
			panic(w.runtime.NewGoError(SecurityError("innerHTML blocked by CSP: inline scripts not allowed")))
		}
		// Check for event handlers in HTML (onclick, onload, etc.)
		if csp != nil && !csp.AllowInlineScript() {
			lowerHTML := strings.ToLower(htmlString)
			eventHandlers := []string{"onclick", "onload", "onerror", "onmouseover", "onmouseout", "onkeydown", "onkeyup", "onkeypress", "onchange", "onsubmit", "onfocus", "onblur"}
			for _, handler := range eventHandlers {
				if strings.Contains(lowerHTML, handler+"=") {
					panic(w.runtime.NewGoError(SecurityError("innerHTML blocked by CSP: inline event handlers not allowed")))
				}
			}
		}
	}

	if err := w.node.SetInnerHTML(htmlString); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	// Return the new innerHTML value
	return w.runtime.ToValue(w.node.InnerHTML())
}

// setTextContent sets the text content of the node.
func (w *JSDOMWrapper) setTextContent(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("textContent setter requires a value argument")))
	}
	text := call.Arguments[0].String()
	w.node.SetTextContent(text)
	// Return the new textContent value
	return w.runtime.ToValue(w.node.GetTextContent())
}

// insertBefore inserts a new child node before an existing child node.
// If referenceNode is null, the new node is appended at the end.
func (w *JSDOMWrapper) insertBefore(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("insertBefore requires a newNode argument")))
	}
	newNodeVal := call.Arguments[0]
	newNodeObj := newNodeVal.ToObject(w.runtime)
	if newNodeObj == nil {
		panic(w.runtime.NewGoError(errors.New("insertBefore newNode must be a Node object")))
	}
	newNodeWrapper := w.getNodeFromJSObject(newNodeObj)
	if newNodeWrapper == nil {
		panic(w.runtime.NewGoError(errors.New("insertBefore newNode must be a valid Node")))
	}
	newNode := newNodeWrapper.node

	if err := newNode.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}

	var referenceNode *NodeDOM
	if len(call.Arguments) >= 2 && !goja.IsNull(call.Arguments[1]) && !goja.IsUndefined(call.Arguments[1]) {
		refVal := call.Arguments[1]
		refObj := refVal.ToObject(w.runtime)
		if refObj == nil {
			panic(w.runtime.NewGoError(errors.New("insertBefore referenceNode must be a Node object")))
		}
		refWrapper := w.getNodeFromJSObject(refObj)
		if refWrapper == nil {
			panic(w.runtime.NewGoError(errors.New("insertBefore referenceNode must be a valid Node")))
		}
		referenceNode = refWrapper.node
		if err := referenceNode.checkOrigin(); err != nil {
			panic(w.runtime.NewGoError(err))
		}
	}

	if err := w.node.InsertBefore(newNode, referenceNode); err != nil {
		panic(w.runtime.NewGoError(err))
	}

	return w.wrapNode(newNode)
}

// replaceChild replaces an existing child node with a new node.
// Returns the replaced node.
func (w *JSDOMWrapper) replaceChild(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if len(call.Arguments) < 2 {
		panic(w.runtime.NewGoError(errors.New("replaceChild requires newNode and oldNode arguments")))
	}
	newNodeVal := call.Arguments[0]
	newNodeObj := newNodeVal.ToObject(w.runtime)
	if newNodeObj == nil {
		panic(w.runtime.NewGoError(errors.New("replaceChild newNode must be a Node object")))
	}
	newNodeWrapper := w.getNodeFromJSObject(newNodeObj)
	if newNodeWrapper == nil {
		panic(w.runtime.NewGoError(errors.New("replaceChild newNode must be a valid Node")))
	}
	newNode := newNodeWrapper.node

	oldNodeVal := call.Arguments[1]
	oldNodeObj := oldNodeVal.ToObject(w.runtime)
	if oldNodeObj == nil {
		panic(w.runtime.NewGoError(errors.New("replaceChild oldNode must be a Node object")))
	}
	oldNodeWrapper := w.getNodeFromJSObject(oldNodeObj)
	if oldNodeWrapper == nil {
		panic(w.runtime.NewGoError(errors.New("replaceChild oldNode must be a valid Node")))
	}
	oldNode := oldNodeWrapper.node

	if err := newNode.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if err := oldNode.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}

	replacedNode, err := w.node.ReplaceChild(newNode, oldNode)
	if err != nil {
		panic(w.runtime.NewGoError(err))
	}

	return w.wrapNode(replacedNode)
}

// cloneNode creates a copy of the node.
// If deep is true, it also clones all descendants.
func (w *JSDOMWrapper) cloneNode(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	deep := false
	if len(call.Arguments) >= 1 {
		deep = call.Arguments[0].ToBoolean()
	}
	clonedNode := w.node.CloneNode(deep)
	return w.wrapNode(clonedNode)
}

// getStyle returns the computed style object for this element.
func (w *JSDOMWrapper) getStyle(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	if w.node.Style == nil {
		return goja.Null()
	}
	styleObj := w.runtime.NewObject()
	style := w.node.Style
	if style.Color != nil {
		styleObj.Set("color", fmt.Sprintf("rgba(%d, %d, %d, %f)",
			int(style.Color.R*255), int(style.Color.G*255), int(style.Color.B*255), style.Color.A))
	}
	if style.BackgroundColor != nil {
		styleObj.Set("backgroundColor", fmt.Sprintf("rgba(%d, %d, %d, %f)",
			int(style.BackgroundColor.R*255), int(style.BackgroundColor.G*255), int(style.BackgroundColor.B*255), style.BackgroundColor.A))
	}
	if style.FontSize != 0 {
		styleObj.Set("fontSize", fmt.Sprintf("%fpx", style.FontSize))
	}
	if style.FontWeight != 0 {
		styleObj.Set("fontWeight", strconv.Itoa(style.FontWeight))
	}
	if style.FontFamily != "" {
		styleObj.Set("fontFamily", style.FontFamily)
	}
	if style.Display != "" {
		styleObj.Set("display", style.Display)
	}
	if style.Position != "" {
		styleObj.Set("position", style.Position)
	}
	if style.Width != 0 {
		styleObj.Set("width", fmt.Sprintf("%fpx", style.Width))
	}
	if style.Height != 0 {
		styleObj.Set("height", fmt.Sprintf("%fpx", style.Height))
	}
	return styleObj
}

// getTextContentProp is the getter for the textContent property.
func (w *JSDOMWrapper) getTextContentProp(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	return w.runtime.ToValue(w.node.GetTextContent())
}

// getInnerHTMLProp is the getter for the innerHTML property.
func (w *JSDOMWrapper) getInnerHTMLProp(call goja.FunctionCall) goja.Value {
	if err := w.node.checkOrigin(); err != nil {
		panic(w.runtime.NewGoError(err))
	}
	return w.runtime.ToValue(w.node.InnerHTML())
}

// wrapNode wraps a NodeDOM in a JSDOMWrapper and returns a JS object.
func (w *JSDOMWrapper) wrapNode(node *NodeDOM) goja.Value {
	wrapper := &JSDOMWrapper{
		node:      node,
		runtime:   w.runtime,
		windowCtx: w.windowCtx,
	}
	obj := w.runtime.NewObject()
	obj.Set("nodeType", node.Type)
	obj.Set("tagName", strings.ToUpper(node.Element))
	// Define textContent with getter/setter
	obj.DefineAccessorProperty("textContent", w.runtime.ToValue(wrapper.getTextContentProp), w.runtime.ToValue(wrapper.setTextContent), 0, 0)
	// Define innerHTML with getter/setter
	obj.DefineAccessorProperty("innerHTML", w.runtime.ToValue(wrapper.getInnerHTMLProp), w.runtime.ToValue(wrapper.setInnerHTML), 0, 0)
	obj.Set("id", node.Attr("id"))
	obj.Set("className", node.Attr("class"))
	obj.Set("querySelector", wrapper.querySelector)
	obj.Set("querySelectorAll", wrapper.querySelectorAll)
	obj.Set("getAttribute", wrapper.getAttribute)
	obj.Set("setAttribute", wrapper.setAttribute)
	obj.Set("removeAttribute", wrapper.removeAttribute)
	obj.Set("appendChild", wrapper.appendChild)
	obj.Set("removeChild", wrapper.removeChild)
	obj.Set("insertBefore", wrapper.insertBefore)
	obj.Set("replaceChild", wrapper.replaceChild)
	obj.Set("cloneNode", wrapper.cloneNode)
	obj.Set("parentNode", wrapper.getParentNode)
	obj.Set("childNodes", wrapper.getChildNodes)
	obj.Set("firstChild", wrapper.getFirstChild)
	obj.Set("lastChild", wrapper.getLastChild)
	obj.Set("nextSibling", wrapper.getNextSibling)
	obj.Set("previousSibling", wrapper.getPreviousSibling)
	obj.Set("style", wrapper.getStyle)
	obj.SetSymbol(nodeWrapperSymbol, wrapper) // Store wrapper reference for retrieval
	return obj
}

// getNodeFromJSObject extracts a JSDOMWrapper from a JS object.
func (w *JSDOMWrapper) getNodeFromJSObject(obj *goja.Object) *JSDOMWrapper {
	if obj == nil {
		return nil
	}
	val := obj.GetSymbol(nodeWrapperSymbol)
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}
	if wrapper, ok := val.Export().(*JSDOMWrapper); ok {
		return wrapper
	}
	return nil
}

// NodeDOM helper methods for child management

// appendChildNode adds a child node to this node.
func (node *NodeDOM) appendChildNode(child *NodeDOM) {
	child.Parent = node
	child.Document = node.Document
	child.WindowCtx = node.WindowCtx
	if len(node.Children) > 0 {
		lastChild := node.Children[len(node.Children)-1]
		lastChild.NextSibling = child
		child.PrevSibling = lastChild
	} else {
		node.FirstChild = child
	}
	node.Children = append(node.Children, child)
}

// removeChildNode removes a child node from this node.
func (node *NodeDOM) removeChildNode(child *NodeDOM) {
	for i, c := range node.Children {
		if c == child {
			if child.PrevSibling != nil {
				child.PrevSibling.NextSibling = child.NextSibling
			}
			if child.NextSibling != nil {
				child.NextSibling.PrevSibling = child.PrevSibling
			}
			if node.FirstChild == child {
				node.FirstChild = child.NextSibling
			}
			node.Children = append(node.Children[:i], node.Children[i+1:]...)
			child.Parent = nil
			child.PrevSibling = nil
			child.NextSibling = nil
			break
		}
	}
}

// JSWindowWrapper methods

// getLocation returns the location object for the window.
func (w *JSWindowWrapper) getLocation(call goja.FunctionCall) goja.Value {
	locObj := w.runtime.NewObject()
	origin := w.windowCtx.GetOrigin()
	if origin != nil {
		locObj.Set("href", origin.String())
		locObj.Set("origin", origin.Scheme+"://"+origin.Host)
		locObj.Set("protocol", origin.Scheme+":")
		locObj.Set("host", origin.Host)
		locObj.Set("hostname", origin.Hostname())
		locObj.Set("port", origin.Port())
		locObj.Set("pathname", "/")
		locObj.Set("search", "")
		locObj.Set("hash", "")
	} else {
		locObj.Set("href", "about:blank")
		locObj.Set("origin", "null")
		locObj.Set("protocol", "")
		locObj.Set("host", "")
		locObj.Set("hostname", "")
		locObj.Set("port", "")
		locObj.Set("pathname", "/")
		locObj.Set("search", "")
		locObj.Set("hash", "")
	}
	locObj.Set("reload", func(call goja.FunctionCall) goja.Value {
		w.windowCtx.EmitEvent("reload", nil)
		return goja.Undefined()
	})
	return locObj
}

// getLocalStorage returns the localStorage object for this window's origin.
func (w *JSWindowWrapper) getLocalStorage(call goja.FunctionCall) goja.Value {
	return w.getStorage("localStorage")
}

// getSessionStorage returns the sessionStorage object for this window's origin.
func (w *JSWindowWrapper) getSessionStorage(call goja.FunctionCall) goja.Value {
	return w.getStorage("sessionStorage")
}

// getStorage returns a Storage object for the given type.
func (w *JSWindowWrapper) getStorage(storageType string) goja.Value {
	storageObj := w.runtime.NewObject()
	var storageMap map[string]string
	if storageType == "localStorage" {
		if val, ok := w.windowCtx.GetInputState("localStorage"); !ok || val == nil {
			storageMap = make(map[string]string)
			w.windowCtx.SetInputState("localStorage", storageMap)
		} else {
			storageMap = val.(map[string]string)
		}
	} else {
		if val, ok := w.windowCtx.GetInputState("sessionStorage"); !ok || val == nil {
			storageMap = make(map[string]string)
			w.windowCtx.SetInputState("sessionStorage", storageMap)
		} else {
			storageMap = val.(map[string]string)
		}
	}
	storageObj.Set("length", len(storageMap))
	storageObj.Set("key", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Null()
		}
		index := int(call.Arguments[0].ToInteger())
		keys := make([]string, 0, len(storageMap))
		for k := range storageMap {
			keys = append(keys, k)
		}
		if index < 0 || index >= len(keys) {
			return goja.Null()
		}
		return w.runtime.ToValue(keys[index])
	})
	storageObj.Set("getItem", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Null()
		}
		key := call.Arguments[0].String()
		if val, ok := storageMap[key]; ok {
			return w.runtime.ToValue(val)
		}
		return goja.Null()
	})
	storageObj.Set("setItem", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(w.runtime.NewGoError(errors.New("setItem requires key and value arguments")))
		}
		key := call.Arguments[0].String()
		value := call.Arguments[1].String()
		storageMap[key] = value
		storageObj.Set("length", len(storageMap))
		return goja.Undefined()
	})
	storageObj.Set("removeItem", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		key := call.Arguments[0].String()
		delete(storageMap, key)
		storageObj.Set("length", len(storageMap))
		return goja.Undefined()
	})
	storageObj.Set("clear", func(call goja.FunctionCall) goja.Value {
		for k := range storageMap {
			delete(storageMap, k)
		}
		storageObj.Set("length", 0)
		return goja.Undefined()
	})
	return storageObj
}

// === JSWindowWrapper methods for event handling and messaging ===

// addEventListener adds an event listener to the window.
func (w *JSWindowWrapper) addEventListener(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(w.runtime.NewGoError(errors.New("addEventListener requires type and listener arguments")))
	}
	eventType := call.Arguments[0].String()
	listener := call.Arguments[1]
	if goja.IsNull(listener) || goja.IsUndefined(listener) {
		return goja.Undefined()
	}
	listenerFunc, ok := goja.AssertFunction(listener)
	if !ok {
		panic(w.runtime.NewGoError(errors.New("addEventListener listener must be a function")))
	}
	w.windowCtx.RegisterEvent(eventType, func(data interface{}) {
		listenerFunc(goja.Undefined(), w.runtime.ToValue(data))
	})
	return goja.Undefined()
}

// removeEventListener removes an event listener from the window.
func (w *JSWindowWrapper) removeEventListener(call goja.FunctionCall) goja.Value {
	// Simplified implementation - in a real browser, you'd track specific listeners
	// For now, we just acknowledge the call
	return goja.Undefined()
}

// postMessage implements window.postMessage for cross-origin communication.
func (w *JSWindowWrapper) postMessage(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(w.runtime.NewGoError(errors.New("postMessage requires message and targetOrigin arguments")))
	}
	message := call.Arguments[0].Export()
	targetOrigin := call.Arguments[1].String()

	// Validate targetOrigin - for now, only allow same-origin or "*"
	windowOrigin := w.windowCtx.GetOrigin().String()
	if targetOrigin != "*" && targetOrigin != windowOrigin {
		// In a real implementation, you'd queue the message for the target window
		// For now, we silently drop cross-origin messages
		return goja.Undefined()
	}

	// Create MessageEvent-like object
	eventObj := w.runtime.NewObject()
	eventObj.Set("data", message)
	eventObj.Set("origin", windowOrigin)
	eventObj.Set("source", w.runtime.Get("window")) // self reference

	// Emit message event on this window (for same-origin)
	w.windowCtx.EmitEvent("message", eventObj)

	return goja.Undefined()
}

// fetch implements the Fetch API.
func (w *JSWindowWrapper) fetch(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewGoError(errors.New("fetch requires a resource argument")))
	}
	// Simplified implementation - returns a rejected promise for now
	promise, _, reject := w.runtime.NewPromise()
	reject(w.runtime.NewGoError(errors.New("fetch not yet implemented")))
	return w.runtime.ToValue(promise)
}

// XMLHttpRequest implements the XMLHttpRequest constructor.
func (w *JSWindowWrapper) XMLHttpRequest(call goja.FunctionCall) goja.Value {
	// Return a constructor function that creates XHR objects
	xhrConstructor := func(call goja.FunctionCall) goja.Value {
		xhrObj := w.runtime.NewObject()
		xhrObj.Set("open", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		xhrObj.Set("send", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		xhrObj.Set("setRequestHeader", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		xhrObj.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		xhrObj.Set("readyState", 0)
		xhrObj.Set("status", 0)
		xhrObj.Set("responseText", "")
		return xhrObj
	}
	return w.runtime.ToValue(xhrConstructor)
}
