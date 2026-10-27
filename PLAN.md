# PLAN.md: Firefox WebExtensions-Compatible Browser

## Current State Assessment

**Implemented and Working (Phases 0-7):**
- Core browser engine with modular architecture
- HTML parser (ketchup), CSS parser (mayo), layout engine (bun)
- 2D graphics library (gg), UI toolkit (mustard)
- Network layer (sauce) with HTTP, cookies, TLS
- JavaScript runtime (hotdog) with Goja VM, DOM bindings, Fetch API
- Per-window isolation model with SOP, CORS, CSP
- Cookie jar, localStorage, sessionStorage
- Comprehensive test suite (all tests passing)

**Bugs Resolved:**
- Missing bundled assets (icons, pages) — bundler.go now generates icons.go and pages.go
- Build failures in mustard and sauce modules resolved

**Implementation Gaps (Phase 8):**
- Browser shell (main.go) is minimal — no tabs, address bar, navigation
- WebView module has stub implementations, not integrated with browser shell
- No Ebitengine integration for cross-platform rendering
- Missing platform-specific GPU backends (Direct3D 12, Metal, Vulkan)
- No Android/iOS targets implemented
- No build tags for CGo vs pure Go rendering selection

---

## Phase E7: Browser UI and User-Facing Features

Build the browser shell UI, tab management, and extension integration.

**Status:** Complete (2026-10-26)

### Tasks
- [x] 1. **Implement Tab Bar UI** — Tab display with container indicators, tab operations
- [x] 2. **Build Address Bar** — URL display, autocomplete, security indicator, container indicator
- [x] 3. **Implement Browser Controls** — Back, forward, reload, home, bookmark, extension icons
- [x] 4. **Build Bookmarks System** — chrome.bookmarks APIs, bookmarks sidebar
- [x] 5. **Implement History** — chrome.history APIs, history sidebar, search
- [x] 6. **Build Settings/Preferences Page** — General, privacy, security, extensions, containers
- [x] 7. **Implement Extension Icon in Toolbar** — Extension icons, popups, badges
- [x] 8. **Build New Tab Page** — Shortcuts, search, recently closed, bookmarks, container selector

### Dependencies
- Core engine (already implemented)
- UI toolkit (mustard) enhancements for browser-specific components
- Ebitengine integration for cross-platform rendering

### Deliverables
- Full-featured browser shell with tabs, navigation, bookmarks
- Extension integration in UI
- Settings and preferences management
- New tab page

---

## Phase E0: Extension Architecture and Manifest Parsing

Establish the foundation for extension execution, discovery, and lifecycle management.

**Status:** Complete (2026-10-27)

### Tasks
- [x] 1. **Define Extension Struct** — Create data structure for manifest, ID, version, permissions, background script VM state, content script registry, storage partition reference, icons/metadata
- [x] 2. **Implement Manifest Parser** — Parse manifest.json (v2 and v3), validate required fields, permissions, icons
- [x] 3. **Create Extension Discovery** — Scan extension directories, auto-load on startup
- [x] 4. **Build Extension Lifecycle Manager** — Enable/disable, reload, uninstall; manage background script startup/shutdown
- [x] 5. **Implement Extension Storage Partition** — Per-extension storage namespace for chrome.storage APIs
- [x] 6. **Create Content Script Registry** — Map URL patterns to content scripts
- [x] 7. **Build Extension Sandboxing** — Isolate extension Goja VMs from page VMs

### Dependencies
- Core engine (already implemented)
- Goja VM per-extension isolation (leverage existing per-window VM model)

### Deliverables
- Extension struct and manifest parser in `browser/extensions/` package
- Extension discovery and lifecycle manager
- Extension sandboxing infrastructure

---

## Phase E5: Content Script Injection, Sandbox, and Security

Enforce robust content script sandboxing and secure injection.

**Status:** Complete (2026-10-27)

### Tasks
- [x] 1. **Build Content Script Loader** — Parse manifest content_scripts, URL pattern matching, run_at timing
- [x] 2. **Implement Content Script Sandbox** — Isolated Goja VM with restricted global access
- [x] 3. **Implement Message Passing with Validation** — Content script ↔ background routing with origin validation
- [x] 4. **Build CSP for Content Scripts** — Enforce manifest content_security_policy
- [x] 5. **Implement Dynamic Content Script Loading** — chrome.scripting.executeScript with ISOLATED/MAIN world support

### Dependencies
- Phase E0 complete
- URL pattern matching library
- CSP parser

### Deliverables
- Content script loader with URL pattern matching
- Secure sandboxed execution environment
- Validated message routing
- CSP enforcement
- Dynamic script injection

---

## Phase E1: Core WebExtensions APIs

**Status:** Complete (2026-10-27)

Implement the foundational APIs that most extensions rely on: messaging, storage, tabs.

### Tasks
- [x] 1. **Build Message Passing System** — chrome.runtime.sendMessage/onMessage, chrome.tabs.sendMessage/onMessage
- [x] 2. **Implement Storage APIs** — chrome.storage.local/sync/managed with get/set/remove/clear
- [x] 3. **Implement Tabs API** — query, get, update, create, remove, events
- [x] 4. **Implement Windows API (basic)** — getCurrent, getAll, create, remove
- [x] 5. **Build Runtime API** — getURL, getManifest, getPlatformInfo, id, events

### Dependencies
- Phase E0 complete
- Tab manager infrastructure (must be built as part of browser shell)

### Deliverables
- Message routing system with extension ID validation
- Storage APIs backed by per-extension JSON files
- Tabs API integrated with browser shell tab manager
- Runtime API with proper event dispatching

---

## Phase E2: Container Tabs API and Container Management

**Status:** Complete (2026-10-27)

### Tasks
- [x] 1. **Define Container Model** — Named groupings with unique cookie jar, localStorage, sessionStorage
- [x] 2. **Build Container Storage Isolation** — chrome.contextualIdentities APIs
- [x] 3. **Extend Tabs API for Containers** — cookieStoreId parameter, tab container metadata
- [x] 4. **Implement Container-Aware Storage** — Partition by container ID + origin
- [x] 5. **Build Container Cookie Management** — chrome.cookies APIs with container filtering
- [x] 6. **Create Container UI Indicators** — Tab color/icon badges, address bar indicator

### Dependencies
- Phase E1 complete
- Enhanced cookie jar supporting partitioning by container ID
- Tab manager supporting container association

### Deliverables
- Container model and storage isolation
- chrome.contextualIdentities API
- Container-aware tabs and cookies APIs
- UI indicators for container tabs

---

## Phase E3: Extended WebExtensions APIs

**Status:** Complete (2026-10-27)

Add advanced APIs for content manipulation and UI customization.

### Tasks
- [x] 1. **Implement Scripting API** — executeScript, insertCSS, removeCSS, dynamic registration
- [x] 2. **Implement Context Menus API** — create, update, remove, removeAll, onClicked
- [x] 3. **Implement Permissions Request & Grants** — request, contains, getAll, events
- [x] 4. **Implement Alarms API** — create, get, getAll, clear, onAlarm
- [x] 5. **Implement i18n API** — getMessage, getAcceptLanguages

### Dependencies
- Phase E1 complete
- Content script injection infrastructure (Phase E5)
- UI integration for context menus and permissions dialogs

### Deliverables
- Scripting API with isolated Goja VM execution
- Context menus integrated with browser UI
- Permission management system
- Alarm scheduling
- i18n localization support

---

## Phase E4: Advanced APIs

**Status:** Complete (2026-10-27)

Implement powerful APIs for network interception, file handling, and user notifications.

### Tasks
- [x] 1. **Implement WebRequest API (read-only)** — onBeforeRequest, onBeforeSendHeaders, onHeadersReceived, onCompleted, onErrorOccurred
- [x] 2. **Implement Notifications API** — create, clear, getAll, events
- [x] 3. **Implement Downloads API** — download, cancel, pause/resume, search, erase, events
- [x] 4. **Implement Cookies API (extended)** — container filtering, getAllCookieStores

### Dependencies
- Phase E1 complete
- Network layer instrumentation (sauce module)
- OS notification integration

### Deliverables
- WebRequest event observers with complete request metadata
- Desktop notification integration
- Download management with OS file system integration
- Extended cookies API

---

## Phase E6: Extension Distribution, Installation, and Updates

Build user-facing installation flow and automatic update mechanisms.

### Tasks
1. **Create Extension Package Format** — .xpi (ZIP) support, package validation, checksum verification
2. **Build Installation UI** — Extension metadata display, permission confirmation
3. **Implement Extension Permissions Prompt** — Human-readable permission descriptions
4. **Build Extension Updates Mechanism** — Update URL checking, download, verify, migrate
5. **Create Extension Uninstall Flow** — Cleanup storage and files
6. **Build Extension Info/Settings Page** — Metadata, storage usage, enable/disable

### Dependencies
- Phase E0 complete
- ZIP archive handling library
- Browser shell UI infrastructure

### Deliverables
- Extension package validation and installation
- Permission prompt UI
- Automatic update mechanism
- Extension management UI

---

## Phase E8: Performance, Testing, and Stability

Ensure the browser is robust, performant, and thoroughly tested.

### Tasks
1. **Build Automated Test Suite** — Unit tests, integration tests, E2E tests, container isolation tests
2. **Implement Memory Profiling** — Memory monitoring per tab/window, leak detection
3. **Add Performance Metrics** — Page load time, paint metrics, extension startup time
4. **Build Debugging Tools** — Extension inspector, background/content script console, network tab
5. **Implement Crash Recovery** — Extension crash handling, tab/container restoration
6. **Add Error Logging and Reporting** — JS errors, permission denials, CSP violations
7. **Optimize Rendering** — Batch DOM mutations, dirty flags, layer caching
8. **Optimize Network** — Connection pooling, DNS caching, TLS session reuse

### Dependencies
- All previous phases
- Profiling and benchmarking libraries
- Debugging tool infrastructure

### Deliverables
- Comprehensive test suite
- Memory and performance profiling tools
- Debugging and developer tools
- Optimized rendering and network stack

---

## Phase E9: Release Preparation and Rollout

Prepare for public release and establish distribution channels.

### Tasks
1. **Create Release Notes and Documentation** — API documentation, guides, troubleshooting
2. **Build Extension Gallery/Marketplace (Optional MVP)** — Curated extension listing
3. **Implement Telemetry (Optional)** — Opt-in, privacy-respecting telemetry
4. **Create Installer** — Single executable installer for Windows, macOS, Linux
5. **Build Auto-Update Infrastructure** — Version checking, staged rollout
6. **Create Communication Channels** — Website, GitHub, community forum
7. **Establish Support Process** — Bug tracker, FAQ, email support
8. **Prepare Security Advisory Process** — Responsible disclosure, security advisories

### Dependencies
- All previous phases
- CI/CD pipeline
- Website and documentation hosting

### Deliverables
- Complete documentation and release notes
- Installer packages for all platforms
- Auto-update mechanism
- Community support infrastructure

---

## Execution Order and Milestones

**Milestone 1: Browser Shell (E7)**
- Browser UI with tabs, address bar, navigation, bookmarks
- New tab page, settings, extension toolbar icons
- Deliverable: Functional browser shell with UI, ready for extension integration

**Milestone 2: Foundation (E0 + E5)**
- Extension architecture, manifest parsing, sandboxing
- Content script injection and security model
- Deliverable: Extensions can be loaded, parsed, and injected securely

**Milestone 3: Core APIs (E1)**
- Messaging, storage, tabs, windows, runtime APIs
- Deliverable: Extensions can communicate, store data, and manage tabs

**Milestone 4: Containers (E2)**
- Container tabs API and storage isolation
- Deliverable: Firefox's container feature fully implemented

**Milestone 5: Extended APIs (E3 + E4)**
- Scripting, menus, permissions, alarms, i18n
- WebRequest, downloads, notifications
- Deliverable: Full WebExtensions API surface

**Milestone 6: Distribution (E6)**
- Extension installation, updates, uninstallation
- Deliverable: Users can install and manage extensions

**Milestone 7: Quality (E8)**
- Testing, performance, debugging tools
- Deliverable: Stable, performant browser ready for release

**Milestone 8: Release (E9)**
- Documentation, installers, community infrastructure
- Deliverable: Public release

---

## Technical Risks and Mitigations

**Risk: Ebitengine Integration Complexity**
- Mitigation: Start with simple window/input integration, progressively add GPU features

**Risk: Extension Compatibility**
- Mitigation: Test against popular Firefox extensions early and often

**Risk: Performance**
- Mitigation: Profile continuously, optimize rendering and network stack

**Risk: Security Model**
- Mitigation: Leverage existing SOP/CORS/CSP infrastructure, thorough security testing

---

## Success Criteria

The browser is considered complete when:
1. All major WebExtensions APIs work
2. Container tabs API fully functional with no cross-container leaks
3. Content script sandboxing secure
4. Security hardened (CSP, SOP, certificate validation)
5. Performance acceptable
6. Stability proven in automated testing
7. Top 100 Firefox extensions work without modification
8. Documentation complete
