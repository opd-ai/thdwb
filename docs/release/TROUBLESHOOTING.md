# thdwb Troubleshooting Guide

Common issues and their solutions for thdwb users and developers.

## Build Issues

### "go: cannot find module"

**Cause:** The repository uses `go.work` workspace mode. Running `go build` from
a submodule directory without the workspace context can fail.

**Solution:** Always build from the repository root:

```sh
cd /path/to/thdwb
make build
```

### "undefined: someSymbol" during build

**Cause:** Build tags may be missing. The Ebitengine backend requires the
`ebitengine` build tag when CGO is disabled.

**Solution:**

```sh
# Pure Go build (no CGo)
CGO_ENABLED=0 go build -tags ebitengine -o bin/thdwb ./browser/

# CGo build (OpenGL/GLFW backend)
CGO_ENABLED=1 go build -o bin/thdwb ./browser/
```

### "OpenGL not found" at runtime

**Cause:** The CGo backend requires OpenGL 3.2+ and GLFW 3.3.

**Solution:** Use the pure-Go Ebitengine backend:

```sh
CGO_ENABLED=0 go build -tags ebitengine -o bin/thdwb ./browser/
```

## Runtime Issues

### Blank white page

**Cause:** The page may be using CSS or JavaScript features not yet supported
by thdwb.

**Solution:**
1. Open the developer console (F12 or Ctrl+Shift+I) to check for errors
2. Try a simpler page to verify the browser works
3. Report the issue at https://github.com/danfragoso/thdwb/issues

### Extension fails to install

**Cause:** Invalid `manifest.json`, missing required fields, or unsupported
permissions.

**Solution:**
1. Validate the manifest against the [EXTENSION_API.md](EXTENSION_API.md)
2. Check the browser console for installation errors
3. Ensure the `.xpi` file is a valid ZIP archive

### Container tabs leaking data

**Cause:** Container isolation relies on separate cookie jars and storage
partitions. If containers share the same ID, data may leak.

**Solution:**
1. Verify each container has a unique ID
2. Check that `cookieStoreId` is set correctly in tab creation
3. Restart the browser to reset container state

### High memory usage

**Cause:** Each tab runs its own JavaScript VM and DOM tree.

**Solution:**
1. Close unused tabs
2. Use the memory profiler (developer tools) to identify leaks
3. Restart the browser periodically

## Debugging

### Enable verbose logging

Set the `THDWB_DEBUG` environment variable:

```sh
THDWB_DEBUG=1 ./bin/thdwb
```

### Open developer tools

Press `F12` or `Ctrl+Shift+I` to open the developer tools panel with:
- JavaScript console
- Network monitor
- Extension inspector
- Storage inspector

### Profile memory usage

```sh
./bin/thdwb --settings=./settings.json
```

Then open the memory profiler from the developer tools.

## Getting Help

- File bugs: https://github.com/danfragoso/thdwb/issues
- FAQ: https://github.com/danfragoso/thdwb/wiki/FAQ
- Discussions: https://github.com/danfragoso/thdwb/discussions
- Email: support@thdwb.dev
