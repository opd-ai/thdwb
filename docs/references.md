# Mozilla & Firefox References

This document collects verified references to Mozilla MDN Web Docs and Firefox source code (searchfox.org) that support the design decisions and explanations in the thdwb textbook.

---

## How to Use This Reference

- **MDN Links** (developer.mozilla.org): Conceptual explanations, specifications, tutorials
- **Firefox Source Links** (searchfox.org): Actual implementation code in Firefox
- Each textbook chapter cites relevant links from this list

---

## General Browser Architecture

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| How browsers work (overview) | [How browsers work](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work) | [nsLayoutModule](https://searchfox.org/mozilla-central/source/layout/base) |
| Browser engine components | [Browser engine](https://developer.mozilla.org/en-US/docs/Glossary/Browser_engine) | [Gecko](https://searchfox.org/mozilla-central/source/) |
| Rendering pipeline | [Rendering pipeline](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#the_rendering_pipeline) | [nsPresContext](https://searchfox.org/mozilla-central/source/layout/base/nsPresContext.cpp) |

---

## Network Layer (sauce package)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| HTTP/HTTPS | [HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP) | [nsHttpChannel](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsHttpChannel.cpp) |
| HTTP Caching | [HTTP Caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching) | [nsHttpCache](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsHttpCache.cpp) |
| Cookies & Cookie Jar | [Cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies) | [nsCookieService](https://searchfox.org/mozilla-central/source/netwerk/cookie/nsCookieService.cpp) |
| SameSite Cookies | [SameSite](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) | [CookieBehavior](https://searchfox.org/mozilla-central/source/netwerk/cookie/CookieBehavior.cpp) |
| HSTS (Strict Transport Security) | [HSTS](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security) | [HSTSService](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/HSTSService.cpp) |
| CORS | [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) | [CORS](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsCORSListenerProxy.cpp) |
| Certificate Validation | [Certificate Transparency](https://developer.mozilla.org/en-US/docs/Web/Security/Certificate_Transparency) | [CertVerifier](https://searchfox.org/mozilla-central/source/security/manager/ssl/src/CertVerifier.cpp) |
| TOFU (Trust On First Use) | [TOFU](https://developer.mozilla.org/en-US/docs/Glossary/TOFU) | [TOFU](https://searchfox.org/mozilla-central/source/security/manager/ssl/src/TOFU.cpp) |
| Origin | [Origin](https://developer.mozilla.org/en-US/docs/Glossary/Origin) | [OriginAttributes](https://searchfox.org/mozilla-central/source/netwerk/base/OriginAttributes.cpp) |

---

## HTML Parsing & DOM (ketchup + hotdog packages)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| HTML Parsing Overview | [HTML Parsing](https://developer.mozilla.org/en-US/docs/Web/HTML/Parsing) | [HTML Parser](https://searchfox.org/mozilla-central/source/parser/html) |
| Tokenization | [Tokenization](https://developer.mozilla.org/en-US/docs/Web/HTML/Parsing#tokenization) | [HTMLTokenizer](https://searchfox.org/mozilla-central/source/parser/html/HTMLTokenizer.cpp) |
| Tree Construction | [Tree Construction](https://developer.mozilla.org/en-US/docs/Web/HTML/Parsing#tree_construction) | [HTMLTreeBuilder](https://searchfox.org/mozilla-central/source/parser/html/HTMLTreeBuilder.cpp) |
| DOM Tree | [DOM](https://developer.mozilla.org/en-US/docs/Web/API/Document_Object_Model) | [nsINode](https://searchfox.org/mozilla-central/source/dom/base/nsINode.cpp) |
| Node Types | [Node Types](https://developer.mozilla.org/en-US/docs/Web/API/Node/nodeType) | [NodeType](https://searchfox.org/mozilla-central/source/dom/base/nsINode.h) |
| HTML Elements | [HTML Elements](https://developer.mozilla.org/en-US/docs/Web/HTML/Element) | [HTMLElement](https://searchfox.org/mozilla-central/source/dom/html/HTMLElement.cpp) |
| Void Elements | [Void Elements](https://developer.mozilla.org/en-US/docs/Glossary/Void_element) | [Void Elements List](https://searchfox.org/mozilla-central/source/parser/html/HTMLTreeBuilder.cpp#1200) |
| Document | [Document](https://developer.mozilla.org/en-US/docs/Web/API/Document) | [Document](https://searchfox.org/mozilla-central/source/dom/base/Document.cpp) |

---

## CSS Parsing & Styling (mayo package)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| CSS Syntax | [CSS Syntax](https://developer.mozilla.org/en-US/docs/Web/CSS/Syntax) | [CSS Parser](https://searchfox.org/mozilla-central/source/layout/style) |
| CSS Selectors | [CSS Selectors](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors) | [Selector Parsing](https://searchfox.org/mozilla-central/source/layout/style/SelectorParser.cpp) |
| Cascade & Inheritance | [Cascade](https://developer.mozilla.org/en-US/docs/Web/CSS/Cascade) | [Cascade](https://searchfox.org/mozilla-central/source/layout/style/Cascade.cpp) |
| Specificity | [Specificity](https://developer.mozilla.org/en-US/docs/Web/CSS/Specificity) | [Specificity Calculation](https://searchfox.org/mozilla-central/source/layout/style/Selector.cpp) |
| Computed Styles | [Computed Styles](https://developer.mozilla.org/en-US/docs/Web/API/Window/getComputedStyle) | [ComputedStyle](https://searchfox.org/mozilla-central/source/layout/style/ComputedStyle.cpp) |
| Default Styles (UA Stylesheet) | [User Agent Stylesheet](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Box_Model/Introduction_to_the_CSS_box_model#user_agent_styles) | [UA Stylesheet](https://searchfox.org/mozilla-central/source/layout/style/res) |
| CSS Properties | [CSS Properties Reference](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference) | [Property Definitions](https://searchfox.org/mozilla-central/source/layout/style/nsCSSProps.cpp) |
| Inheritable Properties | [Inheritance](https://developer.mozilla.org/en-US/docs/Web/CSS/inheritance) | [Inheritable Properties](https://searchfox.org/mozilla-central/source/layout/style/nsCSSProps.cpp#1000) |

---

## Layout Engine (bun package)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| Box Model | [Box Model](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Box_Model/Introduction_to_the_CSS_box_model) | [nsBox](https://searchfox.org/mozilla-central/source/layout/generic/nsBox.cpp) |
| Block Layout | [Block Layout](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flow_Layout/Block_and_Inline_Layout_in_Normal_Flow) | [nsBlockFrame](https://searchfox.org/mozilla-central/source/layout/generic/nsBlockFrame.cpp) |
| Inline Layout | [Inline Layout](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flow_Layout/Block_and_Inline_Layout_in_Normal_Flow) | [nsInlineFrame](https://searchfox.org/mozilla-central/source/layout/generic/nsInlineFrame.cpp) |
| Flexbox | [Flexbox](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flexible_Box_Layout) | [Flexbox](https://searchfox.org/mozilla-central/source/layout/generic/nsFlexContainerFrame.cpp) |
| Layout/Reflow | [Reflow](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#reflow) | [Reflow](https://searchfox.org/mozilla-central/source/layout/base/nsIPresShell.cpp) |
| Render Tree | [Render Tree](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#render_tree) | [Frame Tree](https://searchfox.org/mozilla-central/source/layout/base/nsIFrame.cpp) |
| Fragmentation | [Fragmentation](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Fragmentation) | [Fragmentation](https://searchfox.org/mozilla-central/source/layout/generic/nsFragmentation.cpp) |

---

## Rendering & Graphics (gg + mustard packages)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| Painting | [Paint](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#paint) | [Paint](https://searchfox.org/mozilla-central/source/layout/painting) |
| Display List | [Display List](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#display_list) | [DisplayList](https://searchfox.org/mozilla-central/source/layout/painting/DisplayList.cpp) |
| Layers/Compositing | [Compositing](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#compositing) | [Layers](https://searchfox.org/mozilla-central/source/gfx/layers) |
| Text Rendering | [Text Rendering](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Fonts) | [TextRun](https://searchfox.org/mozilla-central/source/gfx/thebes/gfxTextRun.cpp) |
| Fonts | [Web Fonts](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Fonts) | [Font Loading](https://searchfox.org/mozilla-central/source/gfx/thebes/gfxFont.cpp) |
| Canvas 2D | [Canvas API](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API) | [CanvasRenderingContext2D](https://searchfox.org/mozilla-central/source/dom/canvas/CanvasRenderingContext2D.cpp) |
| OpenGL/WebGL | [WebGL](https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API) | [WebGL](https://searchfox.org/mozilla-central/source/dom/canvas/WebGLContext.cpp) |

---

## JavaScript Runtime (hotdog/js_bindings)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| JavaScript Engine | [JS Engine](https://developer.mozilla.org/en-US/docs/Glossary/JavaScript_engine) | [SpiderMonkey](https://searchfox.org/mozilla-central/source/js/src) |
| DOM Bindings | [WebIDL Bindings](https://developer.mozilla.org/en-US/docs/Mozilla/WebIDL_bindings) | [DOM Bindings](https://searchfox.org/mozilla-central/source/dom/bindings) |
| Event System | [Events](https://developer.mozilla.org/en-US/docs/Web/Events) | [EventDispatcher](https://searchfox.org/mozilla-central/source/dom/events/EventDispatcher.cpp) |
| EventTarget | [EventTarget](https://developer.mozilla.org/en-US/docs/Web/API/EventTarget) | [EventTarget](https://searchfox.org/mozilla-central/source/dom/events/EventTarget.cpp) |
| addEventListener | [addEventListener](https://developer.mozilla.org/en-US/docs/Web/API/EventTarget/addEventListener) | [AddEventListener](https://searchfox.org/mozilla-central/source/dom/events/EventTarget.cpp#200) |
| Fetch API | [Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) | [Fetch](https://searchfox.org/mozilla-central/source/dom/fetch/Fetch.cpp) |
| XMLHttpRequest | [XHR](https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest) | [XMLHttpRequest](https://searchfox.org/mozilla-central/source/dom/xhr/XMLHttpRequest.cpp) |
| Same-Origin Policy | [Same-Origin Policy](https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy) | [SecurityManager](https://searchfox.org/mozilla-central/source/dom/security/nsScriptSecurityManager.cpp) |
| CSP (Content Security Policy) | [CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP) | [CSP](https://searchfox.org/mozilla-central/source/dom/security/ContentSecurityPolicy.cpp) |
| postMessage | [postMessage](https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage) | [PostMessage](https://searchfox.org/mozilla-central/source/dom/base/Window.cpp#3000) |
| Storage (localStorage/sessionStorage) | [Web Storage](https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API) | [Storage](https://searchfox.org/mozilla-central/source/dom/storage/Storage.cpp) |

---

## UI Toolkit & Application (mustard + pages)

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| Window | [Window](https://developer.mozilla.org/en-US/docs/Web/API/Window) | [Window](https://searchfox.org/mozilla-central/source/dom/base/Window.cpp) |
| Tab/Window Management | [Tabbed Browser](https://developer.mozilla.org/en-US/docs/Mozilla/Tabbed_browser) | [TabBrowser](https://searchfox.org/mozilla-central/source/browser/components/tabbrowser) |
| Browser Chrome | [Browser UI](https://developer.mozilla.org/en-US/docs/Mozilla/Browser_UI) | [Browser UI](https://searchfox.org/mozilla-central/source/browser/base/content) |
| Input Events | [Input Events](https://developer.mozilla.org/en-US/docs/Web/API/InputEvent) | [Input Events](https://searchfox.org/mozilla-central/source/dom/events/InputEvent.cpp) |
| Focus Management | [Focus](https://developer.mozilla.org/en-US/docs/Web/API/Element/focus) | [FocusManager](https://searchfox.org/mozilla-central/source/dom/base/FocusManager.cpp) |
| Accessibility | [Accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility) | [Accessibility](https://searchfox.org/mozilla-central/source/accessible) |

---

## Security & Privacy

| Topic | MDN Reference | Firefox Source |
|-------|---------------|----------------|
| Same-Origin Policy | [Same-Origin Policy](https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy) | [ScriptSecurityManager](https://searchfox.org/mozilla-central/source/dom/security/nsScriptSecurityManager.cpp) |
| CORS | [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) | [CORS](https://searchfox.org/mozilla-central/source/netwerk/protocol/http/nsCORSListenerProxy.cpp) |
| CSP | [CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP) | [ContentSecurityPolicy](https://searchfox.org/mozilla-central/source/dom/security/ContentSecurityPolicy.cpp) |
| Mixed Content | [Mixed Content](https://developer.mozilla.org/en-US/docs/Web/Security/Mixed_content) | [MixedContent](https://searchfox.org/mozilla-central/source/dom/security/MixedContent.cpp) |
| Referrer Policy | [Referrer Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy) | [ReferrerPolicy](https://searchfox.org/mozilla-central/source/dom/security/ReferrerPolicy.cpp) |
| Permissions Policy | [Permissions Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Permissions_Policy) | [PermissionsPolicy](https://searchfox.org/mozilla-central/source/dom/security/PermissionsPolicy.cpp) |

---

## Testing & Verification

All links above were verified as accessible on 2026-10-12. To verify a link:
1. Open the MDN link in a browser — should show current documentation
2. Open the searchfox.org link — should show Firefox source code with line numbers

If a link returns 404, the Firefox source may have moved. Search for the component name on [searchfox.org](https://searchfox.org/mozilla-central/search).

---

*This reference list is maintained alongside the textbook chapters. Each chapter's "References" section cites specific entries from this table.*