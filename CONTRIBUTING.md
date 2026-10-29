# Contributing to thdwb

Thank you for your interest in contributing to thdwb, the Hotdog Web Browser!
We welcome contributions from the community. This guide explains how to get
started.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). By
participating, you agree to uphold these standards.

## Getting Started

### Prerequisites

- Go 1.21+
- OpenGL 3.2+ capable GPU (for the CGo backend)
- GLFW 3.3 (via `go-gl/glfw`)

### Building

```sh
git clone https://github.com/danfragoso/thdwb.git
cd thdwb
make build
```

### Running Tests

```sh
make test
```

### Running Lint

```sh
make lint
```

## How to Contribute

### Reporting Bugs

File bugs at https://github.com/danfragoso/thdwb/issues. Include:

1. A clear title and description
2. Steps to reproduce
3. Expected vs. actual behavior
4. Your OS, Go version, and thdwb version
5. Screenshots or logs if applicable

### Suggesting Enhancements

Open an issue describing the feature, its motivation, and any alternative
approaches you've considered.

### Submitting Pull Requests

1. Fork the repository and create a feature branch from `master`.
2. Make your changes, following the existing code style.
3. Add or update tests for your changes.
4. Run `make test` and `make lint` to verify.
5. Commit with a descriptive message (see [Commit Messages](#commit-messages)).
6. Open a pull request against `master`.

### Commit Messages

We follow conventional commit prefixes:

| Prefix | Meaning |
|--------|---------|
| `feat(` | New feature |
| `fix(` | Bug fix |
| `docs(` | Documentation only |
| `test(` | Adding or fixing tests |
| `refactor(` | Code refactoring without behavior change |
| `perf(` | Performance improvement |
| `build(` | Build system or CI changes |
| `ci(` | CI configuration changes |

Example: `feat(e9): Add auto-update infrastructure with staged rollouts`

The scope in parentheses refers to the phase or module (e.g., `e9`, `shell`,
`extensions`, `sauce`).

## Development Workflow

thdwb is organized as a Go workspace with multiple modules:

```
browser/     - Browser shell, extensions, gallery, telemetry, autoupdate
sauce/       - Network layer (HTTP, cookies, TLS)
hotdog/      - JavaScript runtime (Goja VM, DOM bindings)
bun/         - Layout engine
mayo/        - CSS parser & style engine
ketchup/     - HTML parser
gg/          - 2D graphics library
mustard/     - UI toolkit
platform/    - Platform abstraction (renderer, window, input)
webview/     - Embeddable WebView layer
profiler/    - Profiling instrumentation
assets/      - Fonts, icons, bundler
pages/       - Example pages/templates
```

Run tests per module:

```sh
cd sauce && CGO_ENABLED=0 go test ./...
```

Or run all tests from the root:

```sh
make test
```

## Community

- Issues: https://github.com/danfragoso/thdwb/issues
- Discussions: https://github.com/danfragoso/thdwb/discussions
- Email: dev@thdwb.dev

## License

By contributing to thdwb, you agree that your contributions will be licensed
under the [MIT License](LICENSE).
