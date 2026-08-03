# ROADMAP.md: Firefox WebExtensions-Compatible Browser

## Vision

Build a **freestanding consumer browser** that depends on the refactored `thdwb` library and achieves **100% Firefox WebExtensions API compatibility**, including the **Container Tabs API** for origin-keyed tab isolation and user privacy.

The browser itself is a thin orchestration layer over `thdwb` that adds extension management, extension sandboxing, and the complete WebExtensions API surface.

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

* **Create Extension Package Format:**
  - Support `.xpi` (ZIP archive) and directory-based loads for development
  - Validate package structure: manifest.json required at root, all referenced paths present
  - Checksum manifest for integrity verification (reject tampered packages)
  - Extract and cache in extension directory with UUID-based folder naming
* **Build Installation UI:**
  - Browser displays extension metadata: name, icon, permissions list, description
  - Prompt user to confirm installation and grant requested permissions
  - On confirmation, extract package, initialize storage, register content scripts, start background script
  - Display installation success; offer to open extension options page
  - **Installation logs all requested permissions for user review**
* **Implement Extension Permissions Prompt:**
  - Manifest lists `permissions` array; prompt user on install
  - Display human-readable permission descriptions (e.g., "Read your browsing history")
  - User can refuse installation entirely or accept all (no granular accept/deny per permission in this phase)
  - Accepted permissions stored in extension profile
* **Build Extension Updates Mechanism:**
  - Extension manifest includes `update_url` field
  - On startup and at intervals, check update manifest (JSON with version info)
  - If newer version available, download, verify signature, extract, and migrate data
  - Migrate extension storage from old version (schema versioning support)
  - Fire `chrome.runtime.onInstalled` event with reason: "update"
  - **Automatic updates only if permission granted in manifest; user can disable per-extension**
* **Create Extension Uninstall Flow:**
  - User right-clicks extension icon → Uninstall
  - Browser prompts to confirm; offer to delete extension data
  - On confirm, stop background script, clear storage partition, remove from registry
  - Delete extension files from disk
* **Build Extension Info/Settings Page:**
  - Show extension name, version, description, permissions granted
  - Display storage usage (local + sync)
  - Option to reload extension (for development)
  - Option to visit extension's homepage/options page
  - Enable/disable toggle
  - Uninstall button

### Phase E6 Completion Checklist
- [ ] .xpi packages extract and validate correctly
- [ ] Installation prompts show permissions and require confirmation
- [ ] Background scripts start after installation
- [ ] Update mechanism detects and downloads new versions
- [ ] Extension storage migrates on update without data loss
- [ ] Uninstall cleans up all extension data
- [ ] Extension info page displays metadata and usage
- [ ] Reload extension button restarts background script
- [ ] Disabled extensions do not execute scripts or handle events
- [ ] Extension signatures validated (if signed)

---

## Phase E7: Browser UI and User-Facing Features

Build the browser shell UI, tab management, and extension integration into browser controls.

* **Implement Tab Bar UI:**
  - Display tabs horizontally with container color/icon indicators
  - Active tab highlighted; inactive tabs dimmed
  - Right-click on tab: close, duplicate, move to other container, mute audio
  - Drag tabs to reorder
  - Tab thumbnails on hover (optional for first release)
  - Long tabs truncate title with ellipsis
  - Close button on hover
* **Build Address Bar:**
  - Display current URL
  - Suggest browser history, bookmarks, search engine results on input
  - Show security indicator: HTTPS (lock icon), HTTP (warning), mixed content
  - Show container indicator (color/icon) for current tab
  - Address bar focus on Ctrl+L (or Cmd+L on macOS)
* **Implement Browser Controls:**
  - Back, forward, reload buttons
  - Home button (configurable homepage)
  - Bookmark button (add/remove from bookmarks)
  - Extension icon (show all installed extensions, access options/settings)
  - Menu button (File, Edit, View, History, Bookmarks, Tools, Help)
* **Build Bookmarks System:**
  - `chrome.bookmarks.create()`, `chrome.bookmarks.get()`, `chrome.bookmarks.getChildren()`
  - `chrome.bookmarks.move()`, `chrome.bookmarks.remove()`, `chrome.bookmarks.update()`
  - Bookmarks sidebar showing folders and items
  - Right-click bookmark: open, open in new tab, open in container, edit, delete
  - Keyboard shortcut Ctrl+B to toggle sidebar
* **Implement History:**
  - `chrome.history.addUrl()` - auto-called on page load
  - `chrome.history.deleteUrl()`, `chrome.history.deleteRange()`
  - `chrome.history.search(query)` - fuzzy search history
  - `chrome.history.onVisited` event
  - History sidebar: list pages with visit count and timestamps
  - Clear history button with date range selector
* **Build Settings/Preferences Page:**
  - General: homepage, default search engine, startup behavior
  - Privacy: tracking prevention (placeholder), cookie policy per container
  - Security: certificate warnings, HTTPS enforcement level
  - Extensions: list extensions with enable/disable, permissions, storage usage
  - Containers: list containers, add/edit/delete, default container for new tabs
  - Advanced: developer tools, network logging, extension debugging
* **Implement Extension Icon in Toolbar:**
  - One icon per installed extension
  - Left-click opens extension popup (if manifest defines `action` with default_popup)
  - Right-click: manage extension, visit options page, disable/uninstall
  - Badge with notification count (if extension sets via `chrome.action.setBadgeText()`)
  - Popups run in isolated Goja VM with access to background script via message passing
* **Build New Tab Page:**
  - Shortcuts to frequently visited sites
  - Search bar (default search engine configurable)
  - Recently closed tabs
  - Bookmarks quick access
  - Container selector (open new tab in specific container)
  - Extensions can customize new tab page (if manifest defines `chrome_url_overrides.newtab`)

### Phase E7 Completion Checklist
- [ ] Tab bar displays all open tabs with correct container colors
- [ ] Tab switching works smoothly and updates address bar
- [ ] Address bar autocomplete suggests history/bookmarks
- [ ] Security indicator shows HTTPS/HTTP status
- [ ] Back/forward buttons work and are disabled when unavailable
- [ ] Bookmarks API works; sidebar shows bookmarks
- [ ] History search finds pages by title and URL
- [ ] Settings page allows configuring homepage, search engine, containers
- [ ] Extension icons display; popups open on click
- [ ] New tab page is functional with container selector
- [ ] Menu bar (File, Edit, View, etc.) standard items work
- [ ] Keyboard shortcuts work (Ctrl+L, Ctrl+T, Ctrl+W, Ctrl+B, etc.)

---

## Phase E8: Performance, Testing, and Stability

Ensure the browser is robust, performant, and thoroughly tested.

* **Build Automated Test Suite:**
  - Unit tests for extension manifest parsing
  - Unit tests for each WebExtensions API (message passing, storage, tabs, etc.)
  - Integration tests: install extension, open page, trigger events, verify behavior
  - End-to-end tests: navigate to real websites, verify rendering, interact with extension
  - Container isolation tests: verify no cookie/storage leaks between containers
  - Content script security tests: verify page cannot access content script globals
  - WebRequest observer tests: verify correct request metadata
* **Implement Memory Profiling:**
  - Monitor memory usage per tab/window
  - Detect memory leaks on long-running sessions
  - Profile garbage collection frequency
  - Set memory limits per extension (kill if exceeded)
* **Add Performance Metrics:**
  - Measure page load time (TTFB, DOM interactive, page complete)
  - Measure time to first paint, first contentful paint
  - Monitor extension startup time
  - Monitor message passing latency
  - Log slow operations (JS execution >100ms, layout recalc >50ms)
* **Build Debugging Tools:**
  - Extension inspector: view manifest, storage, background script logs
  - Background script console (logs from `console.log()` in background)
  - Content script console (logs from scripts injected into pages)
  - Network tab: observe WebRequest events, response times, sizes
  - Storage tab: view extension storage, cookies, localStorage, sessionStorage per container
  - Container debugger: view isolation boundaries, verify no leaks
* **Implement Crash Recovery:**
  - On extension crash, log error and disable extension (don't auto-restart)
  - On browser crash, restore tabs and containers on restart
  - Implement session persistence: save open tabs, URLs, container associations
  - On startup, restore session (offer option to disable auto-restore)
* **Add Error Logging and Reporting:**
  - Log all JS errors in background scripts and content scripts (with stack traces)
  - Log all permission denial events (when extension requests blocked)
  - Log all CSP violations
  - Log all certificate validation failures
  - Errors viewable in extension inspector and browser dev tools
* **Optimize Rendering:**
  - Batch DOM mutations to minimize layout recalculations
  - Use dirty flags per window to avoid full-page relayout on small changes
  - Implement layer caching for non-mutated subtrees
  - Measure and optimize paint frequency
* **Optimize Network:**
  - Connection pooling (share HTTP connections across tabs where SOP allows)
  - DNS caching (cache resolved addresses)
  - TLS session reuse (share session cache across origins in same container)
  - Implement request prioritization (interactive requests before background prefetch)

### Phase E8 Completion Checklist
- [ ] Test suite covers all major WebExtensions APIs
- [ ] Container isolation tests pass with zero leaks
- [ ] Content script security tests verify sandbox enforcement
- [ ] Memory profiling identifies and eliminates leaks
- [ ] Page load metrics logged and accessible
- [ ] Extension inspector shows logs, storage, manifest
- [ ] Browser recovers from crashes and restores session
- [ ] Error logging captures all exceptions with stack traces
- [ ] Rendering optimized; paint frequency <60fps variance
- [ ] Network optimized; connection reuse measured
- [ ] All major websites render without regressions

---

## Phase E9: Release Preparation and Rollout

Prepare for public release and establish distribution channels.

* **Create Release Notes and Documentation:**
  - Document all supported WebExtensions APIs with examples
  - Document container API and how to use containers in extensions
  - Create getting started guide for extension developers
  - Document limitations vs. Firefox (e.g., no Manifest v3 yet)
  - Create troubleshooting guide
  - Publish API reference docs (auto-generated from code)
* **Build Extension Gallery/Marketplace (Optional MVP):**
  - Simple listing of vetted free extensions (curated, not open submission)
  - Extension metadata: icon, name, version, rating, download count
  - Direct link to .xpi download (unsigned for MVP; signature validation skipped if user opts in)
  - Search by category (privacy, productivity, developer tools, etc.)
* **Implement Telemetry (Optional, Privacy-Respecting):**
  - **Entirely opt-in and transparent**
  - Collect: browser version, OS, extension count, most-used extensions (anonymized)
  - Do not collect: browsing history, passwords, personal data, URLs visited
  - User can view/disable telemetry in Settings
  - Publish aggregated telemetry reports
* **Create Installer:**
  - Single executable installer for Windows, macOS, Linux
  - Creates user profile directory
  - Bundles a set of popular free extensions (optional)
  - Creates desktop shortcut and Start Menu entry
  - Registers as handler for `http://`, `https://` protocols
* **Build Auto-Update Infrastructure:**
  - Browser checks for new versions on startup and daily
  - Updates download in background and prompt user to restart
  - Update rollout can be staged (10% day 1, 50% day 2, 100% day 3) to catch regressions
  - Rollback mechanism if critical issues detected post-release
* **Create Communication Channels:**
  - Official website with download link and documentation
  - GitHub repository (public, accept issues and PRs)
  - Community forum or Discord for user support
  - Twitter/social media updates on new releases
  - Email newsletter for major updates (opt-in)
* **Establish Support Process:**
  - Bug tracker (GitHub Issues) for public reporting
  - FAQ page for common issues
  - Email support address for critical security issues
  - Community moderators to help with user questions
* **Prepare Security Advisory Process:**
  - Responsible disclosure program (security researchers can report privately)
  - Security advisories issued for all CVEs within 30 days of fix
  - Extension developers notified of any thdwb library vulnerabilities
  - Security updates released as patches (e.g., v1.0.1) with expedited rollout

### Phase E9 Completion Checklist
- [ ] API documentation complete and published
- [ ] Getting started guide written and tested
- [ ] Release notes detail all changes since previous version
- [ ] Installer tested on Windows, macOS, Linux
- [ ] Auto-update mechanism tested with staged rollout
- [ ] Website live with download link and docs
- [ ] GitHub repository public with contributing guidelines
- [ ] Security advisory process documented
- [ ] Community channels (forum, Discord, Twitter) live
- [ ] Support email address working and monitored

---

## Post-Release Roadmap (Future)

### Phase E10: Manifest v3 and Advanced Sync Features
- Full Manifest v3 support (Service Workers instead of background scripts)
- `chrome.storage.sync` with actual cloud sync (Firebase or similar)
- WebDriver support for extension testing

### Phase E11: Performance and Scale
- Multi-process architecture (extension processes separate from renderer)
- Tab suspension (unload inactive tabs to save memory)
- Lazy-load extensions (don't start all background scripts on startup)
- Extension marketplace with ratings/reviews and automatic security scanning

### Phase E12: Developer Experience
- Extension debugging protocol (remote debugging)
- Hot reload for extension development
- Extension profiler (CPU, memory, network per extension)
- Automated extension testing framework

---

## Success Criteria

The browser is considered **100% Firefox WebExtensions compatible** when:

1. **All major WebExtensions APIs work:** messaging, storage, tabs, windows, bookmarks, history, cookies, contextualIdentities (container tabs)
2. **Container tabs API fully functional:** isolation enforced, no cross-container leaks, UI integration complete
3. **Content script sandboxing secure:** page cannot access content scripts, content scripts cannot access page globals
4. **Security hardened:** CSP enforced, SOP enforced, certificate validation works, HTTPS preferred
5. **Performance acceptable:** page loads comparable to Firefox on same hardware, extension operations <100ms latency
6. **Stability proven:** no crashes on major websites, 99.9% uptime in automated testing
7. **Extension compatibility tested:** top 100 Firefox extensions work without modification
8. **Documentation complete:** API reference, guides, troubleshooting all written and verified

---

## Dependencies and Assumptions

- **thdwb library** must complete all 6 phases (Phase 0–6) with per-window isolation, DOM, layout, JS runtime, and security fully implemented
- **Extension testing** requires real popular extensions from Firefox Add-ons; licensing/legal review needed
- **Performance targets** assume modern hardware (multi-core CPU, 8GB+ RAM); mobile support out of scope for MVP
- **Security model** assumes users understand container isolation is isolation *at the browser level*, not OS-level; users should not assume containers are as strong as separate browser profiles

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