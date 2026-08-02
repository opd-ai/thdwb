# ROADMAP.md: Firefox WebExtensions-Compatible Browser

## Vision

Build a **freestanding consumer browser** that depends on the refactored `thdwb` library and achieves **100% Firefox WebExtensions API compatibility**, including the **Container Tabs API** for origin-keyed tab isolation and user privacy.

The browser itself is a thin orchestration layer over `thdwb` that adds extension management, extension sandboxing, and the complete WebExtensions API surface.

---

## Architecture Overview

### Three-Layer Design

1. **thdwb Library** (per-window isolation, DOM, layout, JS runtime, networking)
2. **Extension Runtime** (manifest parsing, extension sandbox, API bridge, lifecycle)
3. **Browser Shell** (tab manager, UI, extension discovery, user preferences)

### Extension Execution Model

- **Extension processes run in isolated Goja VMs** separate from window VMs
- **Extensions communicate with window contexts via message passing** (`extension → content script → page`)
- **Container tabs API creates logical groupings** with origin-based isolation enforced by thdwb's existing SOP/CORS layer
- **Extension storage partitioned by extension ID and container origin**

---

## Phase E0: Extension Architecture and Manifest Parsing

Establish the foundation for extension execution, discovery, and lifecycle management.

* **Define Extension Struct:** Create data structure to hold:
  - Manifest (v2 and v3 compatible)
  - Extension ID, version, permissions list
  - Background script VM and lifecycle state
  - Content script registry (domain patterns → script paths)
  - Storage partition reference
  - Icon/metadata for browser UI
* **Implement Manifest Parser:** Parse `manifest.json` (v2 and v3); validate required fields, permissions, icons. Support both Firefox and Chromium manifest formats with compatibility layer.
* **Create Extension Discovery:** Scan extension directories (user profiles, system installs); auto-load on startup.
* **Build Extension Lifecycle Manager:** Handle extension enable/disable, reload, uninstall. Manage background script startup/shutdown.
* **Implement Extension Storage Partition:** Create per-extension storage namespace for `chrome.storage.local`, `chrome.storage.sync`, and `chrome.storage.managed`.
* **Create Content Script Registry:** Map URL patterns to content scripts; support `<all_urls>`, domain wildcards, regex patterns.
* **Build Extension Sandboxing:** Isolate extension Goja VMs from page VMs; prevent direct DOM access from extensions.

### Phase E0 Completion Checklist
- [ ] Manifest parser handles v2 and v3 with no errors
- [ ] Extensions auto-discovered from profile directories
- [ ] Background scripts initialize and execute without hanging
- [ ] Extension storage isolated by extension ID
- [ ] Content script registry matches URLs correctly
- [ ] Extension enable/disable toggles cleanly
- [ ] Uninstall removes all extension data
- [ ] Extension VMs cannot access window VMs directly

---

## Phase E1: Core WebExtensions APIs (Messaging, Storage, Tabs)

Implement the foundational APIs that most extensions rely on.

* **Build Message Passing System:**
  - `chrome.runtime.sendMessage()` from content script → background
  - `chrome.runtime.onMessage` listener in background
  - `chrome.tabs.sendMessage()` from background → content script in specific tab
  - `chrome.tabs.onMessage` listener in content script
  - **Message routing validates extension ID and origin; prevents cross-extension messaging unless permitted**
* **Implement Storage APIs:**
  - `chrome.storage.local` (per-extension persistent key-value store)
  - `chrome.storage.sync` (sync across devices; for this consumer build, same as local)
  - `chrome.storage.managed` (read-only, populated by admin)
  - Support `get()`, `set()`, `remove()`, `clear()`, `getBytesInUse()`
  - **Storage keyed by extension ID + container origin**
* **Implement Tabs API:**
  - `chrome.tabs.query()` with filters (active, currentWindow, url, title, status)
  - `chrome.tabs.get(tabId)` - retrieve single tab metadata
  - `chrome.tabs.update(tabId, updateProperties)` - modify URL, active state, pinned state
  - `chrome.tabs.create(createProperties)` - spawn new tab with URL
  - `chrome.tabs.remove(tabId)` - close tab
  - `chrome.tabs.onCreated`, `chrome.tabs.onRemoved`, `chrome.tabs.onUpdated`, `chrome.tabs.onActivated` events
  - **Tab metadata includes container origin; permission checks prevent extensions from accessing tabs outside their container**
* **Implement Windows API (basic):**
  - `chrome.windows.getCurrent()`
  - `chrome.windows.getAll()`
  - `chrome.windows.create()`, `chrome.windows.remove()`
  - **Windows objects include container info; tabs belong to containers**
* **Build Runtime API:**
  - `chrome.runtime.getURL(path)` - resolve extension resource URL
  - `chrome.runtime.getManifest()` - return parsed manifest
  - `chrome.runtime.getPlatformInfo()` - return OS info
  - `chrome.runtime.id` - extension ID
  - `chrome.runtime.onInstalled` event (with reason: install, update, chrome_update)
  - `chrome.runtime.onStartup` event

### Phase E1 Completion Checklist
- [ ] Content script → background message passing works
- [ ] Background → content script message passing works
- [ ] Cross-extension messaging blocked
- [ ] Storage API reads/writes persist across sessions
- [ ] Storage isolated by extension ID
- [ ] tabs.query() returns correct tab list
- [ ] tabs.create() spawns new window instance via thdwb
- [ ] tabs.remove() closes window and cleans up state
- [ ] Tab events fire on create/remove/update/activate
- [ ] Extension can read its manifest
- [ ] runtime events fire at startup and on install

---

## Phase E2: Container Tabs API and Container Management

Implement Firefox's container API for first-class container/origin isolation.

* **Define Container Model:**
  - Container = named grouping with unique cookie jar, localStorage, and session storage
  - Each container has ID, name, color, icon (for UI)
  - Predefined containers: Personal, Work, Banking, Shopping, Web Development (can add custom)
  - **Containers keyed by origin (scheme + domain + port); enforce thdwb's SOP isolation**
* **Build Container Storage Isolation:**
  - `chrome.contextualIdentities.query()` - list all containers
  - `chrome.contextualIdentities.get(cookieStoreId)` - retrieve container metadata
  - `chrome.contextualIdentities.create(details)` - create new container
  - `chrome.contextualIdentities.update(cookieStoreId, details)` - rename, change icon/color
  - `chrome.contextualIdentities.remove(cookieStoreId)` - delete container (wipe its storage)
  - `chrome.contextualIdentities.onCreated`, `chrome.contextualIdentities.onRemoved`, `chrome.contextualIdentities.onUpdated` events
* **Extend Tabs API for Containers:**
  - `chrome.tabs.create(createProperties)` with `cookieStoreId` parameter to open tab in specific container
  - `chrome.tabs.query()` returns `cookieStoreId` on each tab
  - `chrome.tabs.update()` supports moving tab to different container (cookie/storage transfer)
  - `chrome.tabs.onCreated` event includes container ID
* **Implement Container-Aware Storage:**
  - Extend thdwb's cookie jar to partition by container ID in addition to origin
  - Extend localStorage/sessionStorage to use container ID as part of storage key
  - `chrome.storage.local` scoped to extension + container (if extension installed in container)
* **Build Container Cookie Management:**
  - `chrome.cookies.get()`, `chrome.cookies.getAll()` - retrieve cookies for active container
  - `chrome.cookies.set()` - set cookie in active container
  - `chrome.cookies.remove()` - remove cookie from container
  - `chrome.cookies.onChanged` event (include container ID)
  - **No cross-container cookie leaks; enforced at network layer**
* **Create Container UI Indicators:**
  - Tab UI shows container color/icon badge
  - Address bar shows container context on hover
  - New tab dialog allows selecting container

### Phase E2 Completion Checklist
- [ ] contextualIdentities API lists all containers
- [ ] Containers can be created, renamed, deleted
- [ ] Tabs can be opened in specific containers
- [ ] Cookies strictly partitioned by container + origin
- [ ] localStorage partitioned by container + origin
- [ ] sessionStorage partitioned by container + origin
- [ ] Switching tabs shows container indicator
- [ ] Moving tab to different container transfers cookies/storage
- [ ] No cross-container data leakage in test suite
- [ ] Custom containers persist across sessions

---

## Phase E3: Extended WebExtensions APIs (Scripting, Menus, Permissions)

Add advanced APIs for content manipulation and UI customization.

* **Implement Scripting API:**
  - `chrome.scripting.executeScript(injection)` - inject JS into content script context
  - `chrome.scripting.insertCSS(injection)` - inject CSS into page
  - `chrome.scripting.removeCSS(injection)` - remove injected CSS
  - `chrome.scripting.registerContentScripts()` - dynamically register content scripts (Manifest v3)
  - `chrome.scripting.unregisterContentScripts()` - unregister dynamic scripts
  - **Injected scripts run in isolated Goja VM; cannot access page globals**
* **Implement Context Menus API:**
  - `chrome.contextMenus.create(createProperties)` - add menu item on right-click
  - `chrome.contextMenus.update(itemId, updateProperties)` - modify menu item
  - `chrome.contextMenus.remove(itemId)` - remove menu item
  - `chrome.contextMenus.removeAll()` - clear all extension menus
  - `chrome.contextMenus.onClicked` event - handle menu selection
  - Support contexts: page, selection, link, image, editable
* **Implement Permissions Request & Grants:**
  - `chrome.permissions.request(permissions)` - ask user for runtime permission grant
  - `chrome.permissions.contains(permissions)` - check if permission granted
  - `chrome.permissions.getAll()` - list granted permissions
  - `chrome.permissions.onAdded`, `chrome.permissions.onRemoved` events
  - **Permission checks enforce manifest `permissions` list; reject out-of-manifest requests**
* **Implement Alarms API:**
  - `chrome.alarms.create(name, alarmInfo)` - schedule callback
  - `chrome.alarms.get(name)` - retrieve alarm metadata
  - `chrome.alarms.getAll()` - list all active alarms
  - `chrome.alarms.clear(name)` - cancel alarm
  - `chrome.alarms.onAlarm` event - alarm fired
  - **Alarms persist across extension reload but are cleared on uninstall**
* **Implement i18n API:**
  - `chrome.i18n.getMessage(messageName, substitutions)` - lookup localized string
  - `chrome.i18n.getAcceptLanguages(callback)` - get user language preferences
  - Support `_locales/` directory structure (en, fr, de, etc.)

### Phase E3 Completion Checklist
- [ ] executeScript() injects JS into content script VM
- [ ] insertCSS() applies styles to page without leaking to page globals
- [ ] Context menus appear on right-click with extension items
- [ ] Menu clicks fire onClicked with target context (link, image, selection)
- [ ] permissions.request() prompts user and updates manifest
- [ ] Alarms schedule and fire callbacks at correct intervals
- [ ] i18n.getMessage() returns localized strings
- [ ] Content script can call executeScript to inject nested scripts

---

## Phase E4: Advanced APIs (WebRequest, Downloads, Notifications)

Implement powerful APIs for network interception, file handling, and user notifications.

* **Implement WebRequest API (read-only mode):**
  - `chrome.webRequest.onBeforeRequest` - observe requests before send
  - `chrome.webRequest.onBeforeSendHeaders` - observe headers before send
  - `chrome.webRequest.onHeadersReceived` - observe response headers
  - `chrome.webRequest.onCompleted` - observe finished requests
  - `chrome.webRequest.onErrorOccurred` - observe failed requests
  - Request details include: URL, method, headers, tabId, frameId, initiator, type (xhr, image, stylesheet, etc.)
  - **No request modification (for security/stability); read-only observers only**
* **Implement Notifications API:**
  - `chrome.notifications.create(id, options)` - show desktop notification
  - `chrome.notifications.clear(id)` - dismiss notification
  - `chrome.notifications.getAll(callback)` - list active notifications
  - `chrome.notifications.onClicked`, `chrome.notifications.onClosed` events
  - Support notification types: basic, image, list, progress
* **Implement Downloads API:**
  - `chrome.downloads.download(options)` - start file download
  - `chrome.downloads.cancel(downloadId)` - abort download
  - `chrome.downloads.pause(downloadId)`, `chrome.downloads.resume(downloadId)` - pause/resume
  - `chrome.downloads.search(query)` - query download history
  - `chrome.downloads.erase(query)` - remove from history
  - `chrome.downloads.onCreated`, `chrome.downloads.onChanged`, `chrome.downloads.onErased` events
* **Implement Cookies API (extended):**
  - `chrome.cookies.get()`, `chrome.cookies.getAll()` with container filtering
  - `chrome.cookies.set()`, `chrome.cookies.remove()`
  - `chrome.cookies.getAllCookieStores()` - list containers and their associated tabs
  - `chrome.cookies.onChanged` event

### Phase E4 Completion Checklist
- [ ] WebRequest listeners observe network activity
- [ ] Request details include all required fields
- [ ] Notifications display and dismiss correctly
- [ ] Downloads API integrates with OS download manager
- [ ] Download progress updates in real-time
- [ ] Cookies API returns container-filtered results
- [ ] No request modification side-effects

---

## Phase E5: Content Script Injection, Sandbox, and Security

Enforce robust content script sandboxing and secure injection.

* **Build Content Script Loader:**
  - Parse content script entries from manifest
  - Match URLs against `matches`, `exclude_matches`, `include_globs`, `exclude_globs`
  - Support `run_at` timing: document_start, document_end, document_idle
  - Load script source from extension package (validate path in-package)
  - **Inject into isolated Goja VM with restricted global access**
* **Implement Content Script Sandbox:**
  - Content script VM has access to: `chrome.*` APIs, `console`, `fetch`, `XMLHttpRequest`, DOM manipulation functions (`document.querySelector`, `element.innerHTML`, etc.)
  - Content script VM **cannot** access: page variables, page functions, unsandboxed eval
  - DOM mutations from content script respect SOP; cannot mutate cross-origin iframes
  - Page cannot access content script variables
* **Implement Message Passing with Validation:**
  - Content script ↔ Background script message routing validated by extension ID
  - Content script ↔ Page communication via `window.postMessage()` only (not chrome.runtime.sendMessage)
  - **Messages include origin; block mismatched origins**
* **Build CSP for Content Scripts:**
  - Content scripts subject to manifest `content_security_policy`
  - Prevent inline script execution unless explicitly permitted
  - Restrict external resource loads to extension package URIs
* **Implement Dynamic Content Script Loading:**
  - `chrome.scripting.executeScript()` validates injection target (tabId, frameId, world: MAIN or ISOLATED)
  - ISOLATED world = content script VM
  - MAIN world = page globals (dangerous; only if explicitly requested in manifest)
  - **MAIN world injections sandboxed; cannot call back into extension**

### Phase E5 Completion Checklist
- [ ] Content scripts inject at correct timing (document_start, document_end, document_idle)
- [ ] URL pattern matching works for matches, exclude_matches, globs
- [ ] Content script VM cannot access page globals
- [ ] Page cannot access content script variables
- [ ] Message passing validated by extension ID
- [ ] Dynamic script injection respects ISOLATED vs MAIN world separation
- [ ] CSP violations block inline scripts and log
- [ ] Cross-origin iframe mutations blocked

---

## Phase E6: Extension Distribution, Installation, and Updates

Build user-facing installation flow and automatic update mechanisms.

