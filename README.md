![thdwb](https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/thdwb.png)

This is the hotdog web browser project. It's a web browser with its own layout and rendering engine, parsers, and UI toolkit!

It's made from scratch entirely in golang. External dependencies are only OpenGL and GLFW, even go dependencies are kept to a minimum.

The main goal of this project is to learn how web browsers work under the hood by implementing one. The browser is far from stable, spec-compliant, or even really useful, but, I'm slowly working on bringing more features and supporting more sites.

🌭🌭🌭

<img src="https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/scr_1.png"></img>

### Components
- ketchup (html parser and DOM Tree builder)
- mayo (css parser and Render Tree builder)
- mustard (UI Toolkit, events and OpenGL)
- sauce (requests, cache and filesystem)
- bun (css layout calculator)
- [gg](https://github.com/fogleman/gg) (drawing routines and text rendering)

### HTTPS and self-signed certificates

HTTPS requests validate certificates against the system trust store. Expired certificates are always rejected. To explicitly trust a self-signed certificate, inspect its fingerprint first, verify it through a separate trusted channel, then add it to the persistent TOFU store:

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
    fmt.Println(fingerprint) // Verify this fingerprint before trusting the certificate.
    return store.TrustCertificate("example.test", certificate)
}
```

`TOFUStore.List` reports trusted hostnames and fingerprints. Remove a pin with `store.RemoveCertificate("example.test")`; removing is required before trusting a replacement certificate. Pins are stored under the user's OS configuration directory in `thdwb/tofu-certificates.json` with owner-only file permissions.

### Web storage

`localStorage` is shared by windows with the same origin and persisted under the user's OS configuration directory in `thdwb/local-storage.json`. The data file is restricted to the current user. `sessionStorage` remains per-window and is cleared when its window closes.

### Getting started
- Running

  ```sh
  make
  ```
  This command will start the browser and load an example page

- Testing

  ```sh
  make test
  ```
  This command will run all the configurated unit tests

- Building

  ```sh
  make build
  ```
  This command will build the binary version

### Screenshots

<img src="https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/scr_2.png"></img>

<img src="https://raw.githubusercontent.com/danfragoso/thdwb/master/imgs/scr_3.png"></img>
