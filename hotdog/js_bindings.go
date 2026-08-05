package hotdog

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

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
		docObj.Set("body", docWrapper.getBody)
		docObj.Set("documentElement", docWrapper.getDocumentElement)
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
		// Check CSP for 'unsafe-eval' - for now, always block
		panic(runtime.NewGoError(SecurityError("eval() is blocked by CSP")))
	})
	runtime.Set("Function", func(call goja.FunctionCall) goja.Value {
		// Check CSP for 'unsafe-eval' - for now, always block
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
	result := d.doc.DOM.QuerySelector(selector)
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
	results := d.doc.DOM.QuerySelectorAll(selector)
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
	result := d.doc.DOM.GetElementById(id)
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
	results := d.doc.DOM.GetElementsByClassName(className)
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
	results := d.doc.DOM.GetElementsByTagName(tagName)
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
	body := d.doc.GetBody()
	if body == nil {
		return goja.Null()
	}
	return d.wrapNode(body)
}

// getDocumentElement returns the document element (html).
func (d *JSDocumentWrapper) getDocumentElement(call goja.FunctionCall) goja.Value {
	docEl := d.doc.GetDocumentElement()
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
	obj.Set("nodeType", node.Type)
	obj.Set("tagName", strings.ToUpper(node.Element))
	obj.Set("textContent", node.GetTextContent())
	obj.Set("innerHTML", node.InnerHTML())
	obj.Set("id", node.Attr("id"))
	obj.Set("className", node.Attr("class"))
	obj.Set("querySelector", wrapper.querySelector)
	obj.Set("querySelectorAll", wrapper.querySelectorAll)
	obj.Set("getAttribute", wrapper.getAttribute)
	obj.Set("setAttribute", wrapper.setAttribute)
	obj.Set("removeAttribute", wrapper.removeAttribute)
	obj.Set("appendChild", wrapper.appendChild)
	obj.Set("removeChild", wrapper.removeChild)
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
	result := w.node.QuerySelector(selector)
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
	results := w.node.QuerySelectorAll(selector)
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
	obj.Set("textContent", node.GetTextContent())
	obj.Set("innerHTML", node.InnerHTML())
	obj.Set("id", node.Attr("id"))
	obj.Set("className", node.Attr("class"))
	obj.Set("querySelector", wrapper.querySelector)
	obj.Set("querySelectorAll", wrapper.querySelectorAll)
	obj.Set("getAttribute", wrapper.getAttribute)
	obj.Set("setAttribute", wrapper.setAttribute)
	obj.Set("removeAttribute", wrapper.removeAttribute)
	obj.Set("appendChild", wrapper.appendChild)
	obj.Set("removeChild", wrapper.removeChild)
	obj.Set("parentNode", wrapper.getParentNode)
	obj.Set("childNodes", wrapper.getChildNodes)
	obj.Set("firstChild", wrapper.getFirstChild)
	obj.Set("lastChild", wrapper.getLastChild)
	obj.Set("nextSibling", wrapper.getNextSibling)
	obj.Set("previousSibling", wrapper.getPreviousSibling)
	obj.Set("style", wrapper.getStyle)
	return obj
}

// getNodeFromJSObject extracts a JSDOMWrapper from a JS object.
func (w *JSDOMWrapper) getNodeFromJSObject(obj *goja.Object) *JSDOMWrapper {
	// This is a simplified implementation - in a real browser, you'd use internal slots or WeakMap
	// For now, we can't easily retrieve the Go struct from the JS object
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
		if w.windowCtx.GetInputState("localStorage") == nil {
			storageMap = make(map[string]string)
			w.windowCtx.SetInputState("localStorage", storageMap)
		} else {
			storageMap = w.windowCtx.GetInputState("localStorage").(map[string]string)
		}
	} else {
		if w.windowCtx.GetInputState("sessionStorage") == nil {
			storageMap = make(map[string]string)
			w.windowCtx.SetInputState("sessionStorage", storageMap)
		} else {
			storageMap = w.windowCtx.GetInputState("sessionStorage").(map[string]string)
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
