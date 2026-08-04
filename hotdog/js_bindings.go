package hotdog

import (
	"errors"
	"fmt"
	"net/url"
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
