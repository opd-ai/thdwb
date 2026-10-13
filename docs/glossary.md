# Glossary of Browser Terms

This glossary defines all technical terms used in the thdwb documentation. Terms are defined in plain language suitable for a 9th-grade reader. Each term includes a simple analogy and a reference to Mozilla MDN or Firefox source code.

---

## A

**API (Application Programming Interface)**
- **Definition**: A set of rules that lets one piece of software talk to another. Think of it like a menu at a restaurant — you (the customer) tell the waiter (the API) what you want, and the kitchen (the other software) makes it.
- **Analogy**: A universal remote control that works with any TV brand.
- **Reference**: [MDN - API](https://developer.mozilla.org/en-US/docs/Glossary/API)

**ASCII**
- **Definition**: A standard code that assigns numbers to text characters (letters, numbers, symbols). It's how computers store text as numbers.
- **Analogy**: A secret code where A=65, B=66, etc.
- **Reference**: [MDN - ASCII](https://developer.mozilla.org/en-US/docs/Glossary/ASCII)

---

## B

**Block Element**
- **Definition**: An HTML element that starts on a new line and takes up the full width available. Examples: `<div>`, `<p>`, `<h1>`.
- **Analogy**: A moving box that always sits on its own shelf, pushing everything else down.
- **Reference**: [MDN - Block-level elements](https://developer.mozilla.org/en-US/docs/Web/HTML/Block-level_elements)

**Box Model**
- **Definition**: The CSS concept that every element is a rectangular box with four layers: content, padding, border, and margin.
- **Analogy**: A picture in a frame (border) with a mat (padding) hanging on a wall with space around it (margin).
- **Reference**: [MDN - CSS Box Model](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Box_Model/Introduction_to_the_CSS_box_model)

**Browser Engine**
- **Definition**: The core software that transforms HTML, CSS, and JavaScript into what you see on screen. Also called a "layout engine" or "rendering engine."
- **Analogy**: The engine in a car — you don't see it, but it makes everything move.
- **Reference**: [MDN - Browser engine](https://developer.mozilla.org/en-US/docs/Glossary/Browser_engine)

---

## C

**Cache**
- **Definition**: A temporary storage area where the browser keeps copies of files (images, scripts, stylesheets) so it doesn't have to download them again.
- **Analogy**: A pantry where you keep ingredients you use often, so you don't have to go to the store every time.
- **Reference**: [MDN - HTTP Caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching)

**Cascading (in CSS)**
- **Definition**: The process of combining multiple style rules and deciding which one wins when they conflict. "Cascade" means "waterfall" — rules flow down and later ones can override earlier ones.
- **Analogy**: Layering transparent sheets — the top sheet's marks show through.
- **Reference**: [MDN - Cascade and inheritance](https://developer.mozilla.org/en-US/docs/Web/CSS/Cascade)

**CORS (Cross-Origin Resource Sharing)**
- **Definition**: A security system that lets a web page request resources (like data or images) from a different domain than the one it came from.
- **Analogy**: A bouncer at a club checking IDs — only approved guests (origins) get in.
- **Reference**: [MDN - CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)

**CSS (Cascading Style Sheets)**
- **Definition**: A language for describing how HTML elements should look — colors, fonts, spacing, layout, etc.
- **Analogy**: The interior decorator for a house built with HTML (the structure).
- **Reference**: [MDN - CSS](https://developer.mozilla.org/en-US/docs/Web/CSS)

**CSS Selector**
- **Definition**: A pattern that tells the browser which HTML elements to style. Examples: `p` (all paragraphs), `.highlight` (elements with class="highlight"), `#main` (element with id="main").
- **Analogy**: A name tag that says "style everyone wearing a red shirt."
- **Reference**: [MDN - CSS Selectors](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors)

---

## D

**DOM (Document Object Model)**
- **Definition**: A tree-like structure the browser builds from HTML. Each HTML element becomes a "node" in the tree. JavaScript can read and change the DOM to update the page.
- **Analogy**: A family tree where each person is an HTML element, and you can add, remove, or change family members.
- **Reference**: [MDN - DOM](https://developer.mozilla.org/en-US/docs/Web/API/Document_Object_Model)

**DOCTYPE**
- **Definition**: A declaration at the very top of an HTML file that tells the browser which version of HTML to expect. For modern HTML: `<!DOCTYPE html>`.
- **Analogy**: A label on a box saying "This contains HTML5."
- **Reference**: [MDN - DOCTYPE](https://developer.mozilla.org/en-US/docs/Glossary/Doctype)

---

## E

**Element (HTML Element)**
- **Definition**: A building block of an HTML page, written with tags like `<p>`, `<img>`, `<a>`. Elements can contain text, other elements, or be empty.
- **Analogy**: LEGO bricks — each brick is an element, and you snap them together to build a page.
- **Reference**: [MDN - HTML Elements](https://developer.mozilla.org/en-US/docs/Web/HTML/Element)

**Event**
- **Definition**: Something that happens in the browser — a click, a key press, a page finishing loading. JavaScript can "listen" for events and run code in response.
- **Analogy**: A doorbell — when someone presses it (event), the chime rings (your code runs).
- **Reference**: [MDN - Events](https://developer.mozilla.org/en-US/docs/Web/Events)

**Event Listener**
- **Definition**: A function you register to run when a specific event happens on a specific element.
- **Analogy**: A security guard told "watch the front door and call me if anyone enters."
- **Reference**: [MDN - EventTarget.addEventListener()](https://developer.mozilla.org/en-US/docs/Web/API/EventTarget/addEventListener)

---

## F

**Flexbox (Flexible Box Layout)**
- **Definition**: A CSS layout mode that arranges items in a row or column, letting them grow, shrink, and align automatically based on available space.
- **Analogy**: A row of spring-loaded boxes that stretch and squeeze to fill a shelf perfectly.
- **Reference**: [MDN - Flexbox](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flexible_Box_Layout)

**Frame (Animation Frame)**
- **Definition**: A single still image in a sequence. The browser tries to produce 60 frames per second (60 FPS) for smooth animation.
- **Analogy**: A flipbook — each page is a frame, flip fast and it looks like motion.
- **Reference**: [MDN - requestAnimationFrame](https://developer.mozilla.org/en-US/docs/Web/API/window/requestAnimationFrame)

---

## G

**GPU (Graphics Processing Unit)**
- **Definition**: A specialized chip for drawing graphics. Modern browsers use the GPU to speed up rendering.
- **Analogy**: An art studio full of painters working in parallel, vs. one painter (CPU) doing everything.
- **Reference**: [MDN - GPU](https://developer.mozilla.org/en-US/docs/Glossary/GPU)

---

## H

**HTML (HyperText Markup Language)**
- **Definition**: The standard language for creating web pages. It uses "tags" to mark up text as headings, paragraphs, links, images, etc.
- **Analogy**: The skeleton of a webpage — it gives structure but no style.
- **Reference**: [MDN - HTML](https://developer.mozilla.org/en-US/docs/Web/HTML)

**HTTP (HyperText Transfer Protocol)**
- **Definition**: The protocol (rules) for transferring web pages and data between browsers and servers.
- **Analogy**: The postal service for the web — you send a request (letter), get a response (package).
- **Reference**: [MDN - HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP)

**HTTPS**
- **Definition**: HTTP with encryption (TLS/SSL). The "S" stands for "Secure." It prevents eavesdropping and tampering.
- **Analogy**: Sending a letter in a locked steel box instead of a transparent envelope.
- **Reference**: [MDN - HTTPS](https://developer.mozilla.org/en-US/docs/Glossary/HTTPS)

---

## I

**Inline Element**
- **Definition**: An HTML element that does NOT start on a new line and only takes up as much width as its content. Examples: `<span>`, `<a>`, `<strong>`.
- **Analogy**: Words in a sentence — they flow next to each other, not on separate lines.
- **Reference**: [MDN - Inline elements](https://developer.mozilla.org/en-US/docs/Web/HTML/Inline_elements)

**Inheritance (CSS)**
- **Definition**: When a child element automatically gets certain styles from its parent (like font-family, color). Not all properties inherit.
- **Analogy**: A child inheriting their parent's eye color — it happens automatically unless overridden.
- **Reference**: [MDN - Inheritance](https://developer.mozilla.org/en-US/docs/Web/CSS/inheritance)

---

## J

**JavaScript**
- **Definition**: A programming language that runs in the browser. It can change HTML, CSS, respond to events, fetch data, and more.
- **Analogy**: The brain of a webpage — it makes things interactive and alive.
- **Reference**: [MDN - JavaScript](https://developer.mozilla.org/en-US/docs/Web/JavaScript)

**JS Engine (JavaScript Engine)**
- **Definition**: The part of the browser that reads and executes JavaScript code. Examples: V8 (Chrome), SpiderMonkey (Firefox), Goja (thdwb).
- **Analogy**: A translator who reads JavaScript and tells the computer what to do.
- **Reference**: [MDN - JavaScript Engine](https://developer.mozilla.org/en-US/docs/Glossary/JavaScript_engine)

---

## L

**Layout (Reflow)**
- **Definition**: The process of calculating where every element goes on the page — its x, y position, width, and height.
- **Analogy**: An architect measuring and marking where every wall, window, and door goes on a blueprint.
- **Reference**: [MDN - Layout](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#layout)

---

## M

**Margin**
- **Definition**: The outermost layer of the CSS box model — transparent space outside the border that separates elements from each other.
- **Analogy**: The personal space bubble around a person.
- **Reference**: [MDN - margin](https://developer.mozilla.org/en-US/docs/Web/CSS/margin)

**Markup**
- **Definition**: Special codes (tags) added to text to give it structure and meaning. HTML is a markup language.
- **Analogy**: Highlighting and labeling parts of a document: "This is a heading," "This is a quote."
- **Reference**: [MDN - Markup](https://developer.mozilla.org/en-US/docs/Glossary/Markup)

---

## N

**Node (DOM Node)**
- **Definition**: A single item in the DOM tree. Can be an element, text, comment, or document itself.
- **Analogy**: A single branch or leaf on a tree.
- **Reference**: [MDN - Node](https://developer.mozilla.org/en-US/docs/Web/API/Node)

---

## O

**Origin**
- **Definition**: The combination of scheme (http/https), hostname (example.com), and port (80/443). Two URLs have the "same origin" if all three match.
- **Analogy**: Your home address — same street, same number, same apartment = same origin.
- **Reference**: [MDN - Origin](https://developer.mozilla.org/en-US/docs/Glossary/Origin)

---

## P

**Padding**
- **Definition**: The space between an element's content and its border. Part of the CSS box model.
- **Analogy**: The cushioning inside a picture frame between the photo and the frame.
- **Reference**: [MDN - padding](https://developer.mozilla.org/en-US/docs/Web/CSS/padding)

**Paint (Repaint)**
- **Definition**: The process of drawing pixels on screen based on the calculated layout. Includes text, images, colors, borders, shadows.
- **Analogy**: An artist painting a canvas based on the architect's blueprint.
- **Reference**: [MDN - Paint](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#paint)

**Parser**
- **Definition**: A program that reads text (HTML, CSS, JavaScript) and builds a structured representation (tree, stylesheet, AST).
- **Analogy**: A translator reading a foreign language book and writing an outline in your language.
- **Reference**: [MDN - Parser](https://developer.mozilla.org/en-US/docs/Glossary/Parser)

---

## R

**Reflow**
- **Definition**: Another word for "layout" — recalculating positions and sizes when something changes (window resize, DOM change, style change).
- **Analogy**: Re-measuring the whole house because you moved one wall.
- **Reference**: [MDN - Reflow](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#reflow)

**Render Tree**
- **Definition**: A tree combining the DOM and CSS styles — only visible elements, with their computed styles, ready for layout.
- **Analogy**: A filtered guest list: only people who are actually attending (visible), with their outfits (styles) noted.
- **Reference**: [MDN - Render Tree](https://developer.mozilla.org/en-US/docs/Web/Performance/How_browsers_work#render_tree)

**Rendering Engine**
- **Definition**: Same as "browser engine" — the component that draws the page.
- **Reference**: [MDN - Rendering engine](https://developer.mozilla.org/en-US/docs/Glossary/Rendering_engine)

---

## S

**Same-Origin Policy**
- **Definition**: A security rule: scripts from one origin cannot access data from another origin unless explicitly allowed (via CORS).
- **Analogy**: You can't read your neighbor's mail just because it's in the same mailbox.
- **Reference**: [MDN - Same-origin policy](https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy)

**Selector (CSS Selector)**
- **Definition**: See "CSS Selector" above.

**Specificity (CSS)**
- **Definition**: A scoring system that decides which CSS rule wins when multiple rules target the same element. More specific selectors (like IDs) beat less specific ones (like tags).
- **Analogy**: A tiebreaker — "I said red!" "But I said blue with an ID badge!" → ID badge wins.
- **Reference**: [MDN - Specificity](https://developer.mozilla.org/en-US/docs/Web/CSS/Specificity)

**Stylesheet**
- **Definition**: A collection of CSS rules. Can be external (.css file), internal (<style> tag), or inline (style="" attribute).
- **Analogy**: A rulebook for how things should look.
- **Reference**: [MDN - Stylesheet](https://developer.mozilla.org/en-US/docs/Glossary/Stylesheet)

---

## T

**Tag (HTML Tag)**
- **Definition**: The markup syntax: `<tagname>` (opening), `</tagname>` (closing), `<tagname/>` (self-closing).
- **Analogy**: Label stickers you put on things: "FRAGILE", "THIS SIDE UP".
- **Reference**: [MDN - HTML Tags](https://developer.mozilla.org/en-US/docs/Web/HTML/Element)

**Tokenization**
- **Definition**: The first step of parsing — breaking raw text into meaningful chunks called "tokens" (tags, text, comments).
- **Analogy**: Cutting a sentence into individual words and punctuation marks.
- **Reference**: [MDN - Tokenization](https://developer.mozilla.org/en-US/docs/Web/HTML/Parsing#tokenization)

**Tree (Data Structure)**
- **Definition**: A hierarchical structure with a root, branches, and leaves. The DOM is a tree.
- **Analogy**: An actual tree — trunk (root), branches (parent elements), leaves (child elements).
- **Reference**: [MDN - Tree](https://developer.mozilla.org/en-US/docs/Glossary/Tree)

---

## U

**URL (Uniform Resource Locator)**
- **Definition**: The address of a resource on the web. Example: `https://example.com:443/path?query=value#fragment`
- **Analogy**: A street address for a webpage.
- **Reference**: [MDN - URL](https://developer.mozilla.org/en-US/docs/Glossary/URL)

**User Agent**
- **Definition**: The technical term for "browser" (or any software that acts on behalf of a user). The browser sends a "User-Agent" header identifying itself.
- **Analogy**: Your ID card when you visit a website.
- **Reference**: [MDN - User Agent](https://developer.mozilla.org/en-US/docs/Glossary/User_agent)

---

## V

**Viewport**
- **Definition**: The visible area of a web page inside the browser window. Changes when you resize the window or zoom.
- **Analogy**: A window frame — you only see what's inside the frame.
- **Reference**: [MDN - Viewport](https://developer.mozilla.org/en-US/docs/Glossary/Viewport)

---

## W

**Web Standards**
- **Definition**: Official specifications (from W3C, WHATWG, IETF) that define how HTML, CSS, JavaScript, HTTP, etc. should work. Browsers try to follow them.
- **Analogy**: Building codes that all houses must follow.
- **Reference**: [MDN - Web Standards](https://developer.mozilla.org/en-US/docs/Web/Standards)

---

## X

**XPath**
- **Definition**: A query language for selecting nodes in an XML/HTML document. Like CSS selectors but more powerful.
- **Analogy**: GPS coordinates for finding a specific element in the DOM tree.
- **Reference**: [MDN - XPath](https://developer.mozilla.org/en-US/docs/Web/XPath)

---

*This glossary is a living document. As you read the textbook chapters, refer back here for any unfamiliar terms. All definitions are verified against [MDN Web Docs](https://developer.mozilla.org/) and [Firefox Source Code](https://searchfox.org/mozilla-central/source/).*