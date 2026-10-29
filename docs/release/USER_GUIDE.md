# thdwb User Guide

Welcome to thdwb, the Hotdog Web Browser. This guide covers installation,
basic usage, extensions, and settings.

## Installation

### From Source

```sh
# Clone the repository
git clone https://github.com/danfragoso/thdwb.git
cd thdwb

# Build
make build

# Run
./bin/thdwb
```

### Prerequisites

- Go 1.21+
- OpenGL 3.2+ capable GPU (for the CGo backend)
- GLFW 3.3 (via `go-gl/glfw`)

For a pure-Go build without CGo, use the Ebitengine backend:

```sh
CGO_ENABLED=0 go build -tags ebitengine -o bin/thdwb ./browser/
```

## Basic Usage

### Navigation

- Type a URL in the address bar and press Enter to navigate
- Use Back/Forward buttons to navigate history
- Click Reload to refresh the current page
- Click Home to go to your homepage

### Tabs

- Click the `+` button to open a new tab
- Click the `×` button on a tab to close it
- Click a tab to switch to it
- Right-click a tab for container options

### Bookmarks

- Click the star icon in the address bar to bookmark the current page
- Open the bookmarks sidebar from the menu to manage bookmarks

### History

- Open the history sidebar from the menu to view browsing history
- Use the search box in the history sidebar to find previously visited pages

## Extensions

### Installing an Extension

1. Download a `.xpi` or `.zip` extension package
2. Open the Extensions page from the browser menu
3. Click "Install Extension" and select the package file
4. Review the permissions and click "Install"

### Managing Extensions

- Open the Extensions page from the browser menu
- Enable/disable extensions with the toggle switch
- Click "Remove" to uninstall an extension
- Click "Options" to open an extension's options page

### Developing Extensions

thdwb supports Firefox WebExtensions-compatible extensions. Create a
`manifest.json` file and package your extension as a `.xpi` (ZIP) file.

See [EXTENSION_API.md](EXTENSION_API.md) for the full API reference.

## Settings

Open the Settings page from the browser menu to configure:

### General
- Homepage URL
- Window size and HiDPI scaling

### Privacy & Security
- Cookie settings
- Content Security Policy
- Site permissions

### Extensions
- Enable/disable installed extensions
- View extension permissions
- Access extension options

### Containers
- Create and manage container tabs
- Assign colors and icons to containers

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+T` | New tab |
| `Ctrl+W` | Close tab |
| `Ctrl+L` | Focus address bar |
| `Ctrl+R` | Reload |
| `Ctrl+D` | Bookmark page |
| `Ctrl+H` | Open history |
| `Ctrl+B` | Toggle bookmarks |
| `Ctrl+P` | Print |

## Troubleshooting

### Browser won't start

- Make sure you have Go 1.21+ installed
- Ensure your GPU supports OpenGL 3.2+
- Try the pure-Go Ebitengine backend: `CGO_ENABLED=0 go build -tags ebitengine`

### Pages don't render correctly

- thdwb is a prototype browser with limited CSS support
- Some modern web features may not work
- Check the browser console for JavaScript errors

### Extension not working

- Verify the extension's `manifest.json` is valid
- Check that all required permissions are declared
- Open the extension inspector from the developer tools

### Container tabs not isolated

- Ensure each container has a unique color and name
- Container isolation uses separate cookie jars and storage

## Support

- File bugs at: https://github.com/danfragoso/thdwb/issues
- Read the FAQ: https://github.com/danfragoso/thdwb/wiki/FAQ
- Email: support@thdwb.dev
