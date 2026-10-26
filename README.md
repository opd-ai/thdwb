![thdwb](https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/thdwb.png)

# thdwb — The Hotdog Web Browser

**thdwb** is a from-scratch web browser written entirely in Go. It implements its own:
- HTML parser (**ketchup**)
- CSS parser & style engine (**mayo**)
- Layout engine (**bun**) — block, inline, and flexbox
- 2D graphics library (**gg**) — drawing, text shaping, images
- UI toolkit (**mustard**) — windows, widgets, OpenGL/GLFW
- Network layer (**sauce**) — HTTP, cookies, cache, certificates
- JavaScript runtime (**hotdog**) — Goja VM, DOM bindings, Fetch API

External dependencies: primarily Ebitengine, extant third party Go libraries preferred for all critical components.

> **Main goal**: Learn how web browsers work by building one. Not stable, not spec-compliant, not production-ready — but a great learning tool!

🌭🌭🌭

---

## 📚 The thdwb Textbook

This repository includes a **textbook-style documentation** explaining browser internals at a 9th-grade reading level. Each chapter covers one subsystem with analogies, code walkthroughs, and Mozilla/Firefox references.

| Chapter | Topic | Package |
|---------|-------|---------|
| [01. Introduction](docs/01-introduction.md) | Browser architecture, rendering pipeline | — |
| [02. Network Layer](docs/02-network-layer.md) | HTTP, cookies, cache, TLS, CORS | `sauce` |
| [03. HTML Parsing & DOM](docs/03-html-parsing-dom.md) | Tokenization, tree construction, NodeDOM | `ketchup`, `hotdog` |
| [04. CSS Styling](docs/04-css-styling.md) | Parsing, cascade, inheritance, computed styles | `mayo` |
| [05. Layout Engine](docs/05-layout-engine.md) | Block, inline, flexbox layout | `bun` |
| [06. Rendering & Graphics](docs/06-rendering-graphics.md) | gg 2D graphics, mustard UI toolkit | `gg`, `mustard` |
| [07. JavaScript Runtime](docs/07-javascript-runtime.md) | Goja VM, DOM bindings, Fetch, events | `hotdog` |
| [08. UI Toolkit & App](docs/08-ui-toolkit.md) | Tabs, toolbar, window management | `mustard` |
| [Glossary](docs/glossary.md) | All technical terms defined | — |
| [References](docs/references.md) | MDN & Firefox source links | — |

---

## 🖼️ Screenshots

<img src="https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/scr_1.png" width="400">

<img src="https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/scr_2.png" width="400">

<img src="https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/scr_3.png" width="400">

---

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- OpenGL 3.2+ capable GPU
- GLFW 3.3 (via `go-gl/glfw`)

### Running

```sh
make run
```
Starts the browser (builds first with `make build`).

### Testing

The repository is a `go.work` workspace, so tests run per module or across the
whole workspace:

```sh
make test              # Run every module's tests (CGO_ENABLED=0)
go test work           # Run all tests in the workspace (from the repo root)
go test -race work     # Same, with the race detector (requires CGO)
make coverage          # Per-module and combined coverage reports
```

To test a single module, `cd` into it (e.g. `cd mustard && go test ./...`).

### Building

```sh
make build
```
Produces a binary in the project root (`bin/thdwb`).

---

## 🔐 HTTPS & Self-Signed Certificates

HTTPS requests validate certificates against the system trust store. Expired certificates are always rejected.

To explicitly trust a self-signed certificate (e.g., for local development):

```go
import (
    "context"
    "fmt"
    "github.com/danfragoso/thdwb/sauce"
)

func trustSelfSignedCertificate(ctx context.Context) error {
    store, err := sauce.DefaultTOFUStore()
    if err != nil {
        return err
    }
    certificate, fingerprint, err := sauce.InspectTOFUCertificate(ctx, "example.test:443")
    if err != nil {
        return err
    }
    fmt.Println(fingerprint) // Verify this fingerprint through a trusted channel!
    return store.TrustCertificate("example.test", certificate)
}
```

- `TOFUStore.List` — reports trusted hostnames and fingerprints
- `store.RemoveCertificate("example.test")` — remove a pin (required before trusting a replacement)
- Pins stored in `~/.config/thdwb/tofu-certificates.json` (owner-only permissions)

> 📖 **Learn more**: [Chapter 2: Network Layer](docs/02-network-layer.md#27-security-tls-certificates-hsts-cors)

---

## 💾 Web Storage

- **localStorage** — Shared by windows with the same origin, persisted in `~/.config/thdwb/local-storage.json`
- **sessionStorage** — Per-window, cleared when window closes

> 📖 **Learn more**: [Chapter 7: JavaScript Runtime](docs/07-javascript-runtime.md#713-storage-localstorage--sessionstorage)

---

## 🛡️ Content Security Policy

Documents can enforce `Content-Security-Policy` response headers via `Document.ApplyContentSecurityPolicyHeaders`. The browser checks:
- Inline script/style markup and event handlers
- Script/style/image sources in inserted markup and JS DOM mutations
- Image requests during rendering

Blocked operations return a `SecurityError` and log a console error attributed to the document origin.

> 📖 **Learn more**: [Chapter 7: JavaScript Runtime](docs/07-javascript-runtime.md#715-security-same-origin-policy--csp)

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        thdwb PACKAGES                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐     │
│   │  sauce  │───►│ ketchup │───►│  mayo   │───►│   bun   │     │
│   │ (Net)   │    │ (HTML)  │    │ (CSS)   │    │(Layout) │     │
│   └─────────┘    └────┬────┘    └────┬────┘    └────┬────┘     │
│                      │              │              │           │
│                      ▼              ▼              ▼           │
│               ┌─────────────────────────────────────────┐      │
│               │              hotdog                      │      │
│               │  (DOM Tree + JS Runtime + WindowContext) │      │
│               └────────────────────┬────────────────────┘      │
│                                    │                           │
│                                    ▼                           │
│               ┌─────────────────────────────────────────┐      │
│               │              mustard                     │      │
│               │  (UI Toolkit: Windows, Widgets, OpenGL)  │      │
│               └────────────────────┬────────────────────┘      │
│                                    │                           │
│                                    ▼                           │
│               ┌─────────────────────────────────────────┐      │
│               │                  gg                      │      │
│               │         (2D Graphics & Text)             │      │
│               └─────────────────────────────────────────┘      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🤝 Contributing

This is a learning project! Issues and PRs welcome. Areas needing work:
- HTML5 parsing compliance
- CSS spec compliance
- Flexbox layout (experimental)
- JavaScript ES6+ support
- More web platform APIs

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.
