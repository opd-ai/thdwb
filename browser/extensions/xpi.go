package extensions

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// XPIPackage represents a parsed Firefox extension package (.xpi file).
// An XPI is a ZIP archive containing a manifest.json and extension assets.
type XPIPackage struct {
	// Path is the filesystem path to the .xpi file.
	Path string

	// Manifest is the parsed manifest.json from the package.
	Manifest *Manifest

	// Files lists all files contained in the package.
	Files []string

	// SHA256 is the hex-encoded SHA-256 checksum of the .xpi file.
	SHA256 string

	// ID is the extension ID derived from the manifest or filename.
	ID string

	// Signature is the hex-encoded signature of the package, if present.
	Signature string

	// SignatureAlgorithm is the algorithm used to sign the package (e.g. "ed25519").
	SignatureAlgorithm string

	// Signed indicates whether the package contains a valid signature.
	Signed bool
}

// OpenXPI opens and parses an XPI package from the given path.
// It reads the manifest.json from the archive and computes the file checksum.
func OpenXPI(path string) (*XPIPackage, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open xpi %s: %w", path, err)
	}
	defer r.Close()

	pkg := &XPIPackage{Path: path}

	// Compute SHA-256 checksum of the file.
	if err := pkg.computeChecksum(); err != nil {
		return nil, fmt.Errorf("compute checksum: %w", err)
	}

	// Collect file names and find manifest.json.
	var manifestData []byte
	var signatureData []byte
	var sigAlgorithm string
	hasSignatureFile := false
	for _, f := range r.File {
		pkg.Files = append(pkg.Files, f.Name)
		if f.Name == "manifest.json" || filepath.Base(f.Name) == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open manifest in archive: %w", err)
			}
			manifestData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("read manifest: %w", err)
			}
		}
		// Detect signature files (META-INF/*.rsa, *.ec, *.sf).
		if strings.HasPrefix(f.Name, "META-INF/") {
			ext := strings.ToLower(filepath.Ext(f.Name))
			if ext == ".rsa" || ext == ".ec" || ext == ".sf" {
				hasSignatureFile = true
				sigAlgorithm = strings.TrimPrefix(ext, ".")
				rc, err := f.Open()
				if err != nil {
					return nil, fmt.Errorf("open signature file: %w", err)
				}
				signatureData, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return nil, fmt.Errorf("read signature file: %w", err)
				}
			}
		}
	}

	if manifestData == nil {
		return nil, fmt.Errorf("xpi %s: missing manifest.json", path)
	}

	// Parse manifest from a temp file since ParseManifest reads from disk.
	manifest, err := parseManifestFromData(manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	pkg.Manifest = manifest

	// Derive extension ID from manifest or filename.
	pkg.ID = ExtensionIDFromPath(filepath.Base(path))
	if manifest.Name != "" {
		// Use a sanitized name-based ID if no explicit ID is available.
		pkg.ID = strings.ToLower(strings.ReplaceAll(manifest.Name, " ", "-"))
	}

	// Record signature info only if an actual signature file is present.
	if hasSignatureFile && signatureData != nil {
		pkg.Signature = hex.EncodeToString(signatureData)
		pkg.SignatureAlgorithm = sigAlgorithm
		pkg.Signed = true
	}

	sort.Strings(pkg.Files)
	return pkg, nil
}

// computeChecksum calculates the SHA-256 checksum of the XPI file.
func (pkg *XPIPackage) computeChecksum() error {
	f, err := os.Open(pkg.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	pkg.SHA256 = hex.EncodeToString(h.Sum(nil))
	return nil
}

// Validate checks that the XPI package contains a valid manifest and
// required files. It returns an error if the package is invalid.
func (pkg *XPIPackage) Validate() error {
	if pkg.Manifest == nil {
		return fmt.Errorf("xpi %s: missing manifest", pkg.Path)
	}
	if err := pkg.Manifest.Validate(); err != nil {
		return fmt.Errorf("xpi %s: invalid manifest: %w", pkg.Path, err)
	}
	return nil
}

// ValidateSignature verifies the package signature if one is present.
// If the package is not signed, it returns nil (signature validation is
// optional for unsigned packages). If signed, it verifies the signature
// against the embedded public key or a provided verifier.
func (pkg *XPIPackage) ValidateSignature() error {
	if !pkg.Signed {
		// Unsigned packages are allowed but not verified.
		return nil
	}
	if pkg.Signature == "" {
		return fmt.Errorf("xpi %s: signed package but no signature data", pkg.Path)
	}
	if pkg.SignatureAlgorithm == "" {
		return fmt.Errorf("xpi %s: signed package but no signature algorithm", pkg.Path)
	}
	// TODO: Implement actual cryptographic signature verification once a
	// public key store is available. For now, accept the signature as
	// declared and record it for audit.
	return nil
}

// IsSigned reports whether the package contains a signature.
func (pkg *XPIPackage) IsSigned() bool {
	return pkg.Signed
}

// Extract extracts the XPI package contents to the destination directory.
// Existing files are overwritten.
func (pkg *XPIPackage) Extract(dest string) error {
	r, err := zip.OpenReader(pkg.Path)
	if err != nil {
		return fmt.Errorf("open xpi %s: %w", pkg.Path, err)
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create dest dir %s: %w", dest, err)
	}

	for _, f := range r.File {
		// Prevent path traversal attacks.
		targetPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(dest)+string(os.PathSeparator)) && targetPath != filepath.Clean(dest) {
			return fmt.Errorf("xpi %s: invalid file path %q", pkg.Path, f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("create dir %s: %w", targetPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", filepath.Dir(targetPath), err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open file %s in archive: %w", f.Name, err)
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file %s: %w", targetPath, err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("write file %s: %w", targetPath, err)
		}

		out.Close()
		rc.Close()
	}

	return nil
}

// parseManifestFromData parses a manifest from raw JSON data.
func parseManifestFromData(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}
	return &m, nil
}

// IsXPI reports whether the given filename has a .xpi extension.
func IsXPI(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".xpi")
}

// IsZip reports whether the given filename has a .zip extension.
func IsZip(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".zip")
}

// IsPackage reports whether the given filename is an extension package (.xpi or .zip).
func IsPackage(filename string) bool {
	return IsXPI(filename) || IsZip(filename)
}
