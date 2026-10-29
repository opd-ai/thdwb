# thdwb Release Notes

## Version 0.1.0 — Initial Release

**Release date:** 2026-10-29

thdwb is a from-scratch web browser written entirely in Go. This initial release
implements a complete browser engine with HTML parsing, CSS styling, layout,
rendering, JavaScript execution, networking, and Firefox WebExtensions-compatible
extension support.

### What's New

#### Browser Shell
- Full-featured browser UI with tabs, address bar, navigation controls
- Bookmarks system with sidebar
- Browsing history with search
- Settings/preferences page (general, privacy, security, extensions, containers)
- New tab page with shortcuts, search, recently closed, bookmarks
- Extension toolbar icons with popups and badges

#### WebExtensions Support
- Manifest v2 and v3 parsing and validation
- Extension discovery, lifecycle management (enable/disable/reload/uninstall)
- Per-extension storage partition
- Content script injection with sandboxed Goja VMs
- Message passing between content scripts and background scripts
- Container tabs API with storage isolation

#### Core WebExtensions APIs
- `chrome.runtime` — messaging, manifest info, platform info
- `chrome.tabs` — query, get, update, create, remove, events
- `chrome.windows` — getCurrent, getAll, create, remove
- `chrome.storage` — local/sync/managed with get/set/remove/clear
- `chrome.contextualIdentities` — container management
- `chrome.scripting` — executeScript, insertCSS, removeCSS
- `chrome.contextMenus` — create, update, remove, onClicked
- `chrome.permissions` — request, contains, getAll
- `chrome.alarms` — create, get, getAll, clear, onAlarm
- `chrome.i18n` — getMessage, getAcceptLanguages
- `chrome.webRequest` — read-only request observers
- `chrome.notifications` — create, clear, getAll
- `chrome.downloads` — download, cancel, pause/resume, search, erase
- `chrome.cookies` — extended with container filtering

#### Extension Distribution
- .xpi (ZIP) package support with validation and checksum verification
- Installation UI with permission confirmation
- Human-readable permission descriptions
- Extension update checking, download, verification, and migration
- Extension uninstall flow with storage cleanup
- Extension info/settings page

#### Performance & Stability
- Automated test suite (unit, integration, E2E, container isolation)
- Memory profiling per tab/window with leak detection
- Performance metrics (page load time, paint metrics, extension startup time)
- Debugging tools (extension inspector, console, network tab)
- Crash recovery for extensions, tabs, and containers
- Error logging and reporting (JS errors, CSP violations, permission denials)
- Optimized rendering (batched DOM mutations, dirty flags, layer caching)
- Optimized network (connection pooling, DNS caching, TLS session reuse)

### Known Limitations

- Not production-ready; intended as a learning tool and prototype
- Limited CSS feature support compared to modern browsers
- No Android or iOS targets
- GPU backends limited to OpenGL (CGo) and Ebitengine (pure Go)

### Getting Started

```sh
# Build
make build

# Run
make run

# Run tests
make test
```

See [docs/01-introduction.md](../01-introduction.md) for the full textbook
and [docs/references.md](../references.md) for MDN and Firefox source links.
