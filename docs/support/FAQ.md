# thdwb Frequently Asked Questions

## General

### What is thdwb?

thdwb (The Hotdog Web Browser) is a from-scratch web browser written entirely
in Go. It implements its own HTML parser, CSS parser, layout engine, 2D
graphics library, UI toolkit, network layer, and JavaScript runtime. It also
supports Firefox WebExtensions-compatible extensions.

### Is thdwb production-ready?

No. thdwb is a learning tool and prototype. It is not stable, not fully
spec-compliant, and not production-ready. Use it for experimentation and
learning, not for daily browsing of sensitive or critical sites.

### What platforms does thdwb support?

thdwb builds on Linux, macOS, and Windows. It has two rendering backends:

- **CGo/OpenGL** (default): Uses OpenGL 3.2+ and GLFW 3.3
- **Ebitengine** (pure Go): `CGO_ENABLED=0 go build -tags ebitengine`

There are no Android or iOS targets yet.

## Installation

### How do I build thdwb from source?

```sh
git clone https://github.com/danfragoso/thdwb.git
cd thdwb
make build
./bin/thdwb
```

### How do I build without CGo?

```sh
CGO_ENABLED=0 go build -tags ebitengine -o bin/thdwb ./browser/
```

### Where can I download pre-built installers?

Pre-built installers are not yet available. See the [installer](installer/)
package for building your own Windows, macOS, or Linux installer.

## Usage

### How do I open a new tab?

Click the `+` button in the tab bar, or press `Ctrl+T`.

### How do I bookmark a page?

Click the star icon in the address bar, or press `Ctrl+D`.

### How do I open the developer tools?

Press `F12` or `Ctrl+Shift+I`.

### How do I change the homepage?

Open Settings from the browser menu and set the Homepage URL under General.

## Extensions

### How do I install an extension?

1. Download a `.xpi` or `.zip` extension package
2. Open the Extensions page from the browser menu
3. Click "Install Extension" and select the package
4. Review permissions and click "Install"

### Can I use Firefox extensions?

thdwb supports a Firefox-compatible subset of the WebExtensions API. Many
simple extensions will work, but extensions using unsupported APIs may not.
See [docs/release/EXTENSION_API.md](../release/EXTENSION_API.md) for the
supported API list.

### How do I develop an extension for thdwb?

Create a `manifest.json` file (Manifest V2 or V3) and package your extension
as a `.xpi` (ZIP) file. Use the supported WebExtensions APIs listed in the
API reference.

## Containers

### What are container tabs?

Container tabs isolate browsing data (cookies, localStorage, sessionStorage)
by container. Each container has its own cookie jar and storage partition,
preventing cross-container tracking.

### How do I create a container?

Open Settings > Containers and click "Add Container". Assign a name, color,
and icon. Then open a new tab in that container from the tab bar.

## Troubleshooting

See [docs/release/TROUBLESHOOTING.md](../release/TROUBLESHOOTING.md) for
build issues, runtime issues, and debugging instructions.

## Support

### Where can I get help?

- File bugs: https://github.com/danfragoso/thdwb/issues
- Ask questions: https://github.com/danfragoso/thdwb/discussions
- FAQ: This document
- Email: support@thdwb.dev

### How do I report a security vulnerability?

Email security@thdwb.dev. Do not file public GitHub issues for security
vulnerabilities. See [SECURITY.md](../SECURITY.md) for details.

### How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for the contribution guide,
development workflow, and commit conventions.
