// Package security implements extension marketplace security scanning: it
// scans extension packages for known-vulnerable permissions, suspicious
// patterns, and other security concerns before they are listed or installed.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Severity describes the severity of a security finding.
type Severity int

const (
	// SeverityInfo is an informational finding.
	SeverityInfo Severity = iota
	// SeverityLow is a low-severity finding.
	SeverityLow
	// SeverityMedium is a medium-severity finding.
	SeverityMedium
	// SeverityHigh is a high-severity finding.
	SeverityHigh
	// SeverityCritical is a critical-severity finding.
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Finding describes a single security issue found during scanning.
type Finding struct {
	// RuleID is the identifier of the rule that triggered this finding.
	RuleID string `json:"rule_id"`

	// Severity is the severity of the finding.
	Severity Severity `json:"severity"`

	// Description is a human-readable description of the finding.
	Description string `json:"description"`

	// File is the file within the extension where the issue was found, if any.
	File string `json:"file,omitempty"`

	// Line is the line number within the file, if applicable.
	Line int `json:"line,omitempty"`
}

// ScanResult describes the result of scanning an extension.
type ScanResult struct {
	// ExtensionID is the ID of the scanned extension.
	ExtensionID string `json:"extension_id"`

	// Version is the extension version that was scanned.
	Version string `json:"version"`

	// Findings is the list of security findings.
	Findings []Finding `json:"findings"`

	// Safe reports whether the extension passed security scanning (no
	// high or critical findings).
	Safe bool `json:"safe"`

	// ScannedAt is when the scan was performed.
	ScannedAt string `json:"scanned_at"`
}

// Rule is a security scanning rule.
type Rule interface {
	// ID returns the rule identifier.
	ID() string

	// Scan checks the extension directory for violations and returns
	// findings.
	Scan(extDir string) ([]Finding, error)
}

// Scanner runs security rules against extension packages.
type Scanner struct {
	mu   sync.Mutex
	rules []Rule
}

// NewScanner creates a Scanner with the default set of security rules.
func NewScanner() *Scanner {
	return &Scanner{
		rules: []Rule{
			&dangerousPermissionsRule{},
			&evalUsageRule{},
			&remoteCodeRule{},
		},
	}
}

// AddRule registers a custom security rule.
func (s *Scanner) AddRule(rule Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
}

// Scan scans the extension directory at extDir and returns a ScanResult.
func (s *Scanner) Scan(extID, version, extDir string) (*ScanResult, error) {
	s.mu.Lock()
	rules := append([]Rule(nil), s.rules...)
	s.mu.Unlock()

	var findings []Finding
	for _, rule := range rules {
		ruleFindings, err := rule.Scan(extDir)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.ID(), err)
		}
		findings = append(findings, ruleFindings...)
	}

	safe := true
	for _, f := range findings {
		if f.Severity >= SeverityHigh {
			safe = false
			break
		}
	}

	return &ScanResult{
		ExtensionID: extID,
		Version:     version,
		Findings:    findings,
		Safe:        safe,
		ScannedAt:   "now",
	}, nil
}

// dangerousPermissionsRule flags extensions that request dangerous
// permissions without justification.
type dangerousPermissionsRule struct{}

func (r *dangerousPermissionsRule) ID() string { return "dangerous-permissions" }

var dangerousPermissions = map[string]string{
	"webRequestBlocking": "can block and modify network requests",
	"tabs":               "can read tab URLs and titles",
	"webRequest":         "can observe network requests",
	"cookies":            "can read and modify cookies",
	"downloads":          "can download and save files",
	"nativeMessaging":    "can communicate with native applications",
}

func (r *dangerousPermissionsRule) Scan(extDir string) ([]Finding, error) {
	manifestPath := filepath.Join(extDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil // no manifest to scan
	}

	var manifest struct {
		Permissions []string `json:"permissions"`
		OptionalPermissions []string `json:"optional_permissions,omitempty"`
	}
	if err := unmarshalJSON(data, &manifest); err != nil {
		return nil, err
	}

	var findings []Finding
	allPerms := append(manifest.Permissions, manifest.OptionalPermissions...)
	for _, perm := range allPerms {
		if desc, ok := dangerousPermissions[perm]; ok {
			findings = append(findings, Finding{
				RuleID:      r.ID(),
				Severity:    SeverityMedium,
				Description: fmt.Sprintf("permission %q allows %s", perm, desc),
			})
		}
	}
	return findings, nil
}

// evalUsageRule flags extensions that use eval() or similar dangerous JS
// functions in their scripts.
type evalUsageRule struct{}

func (r *evalUsageRule) ID() string { return "eval-usage" }

func (r *evalUsageRule) Scan(extDir string) ([]Finding, error) {
	var findings []Finding

	// Walk the extension directory looking for JS files.
	err := filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".js") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "eval(") ||
				strings.Contains(line, "setTimeout(") && strings.Contains(line, "\"") ||
				strings.Contains(line, "new Function(") {
				findings = append(findings, Finding{
					RuleID:      r.ID(),
					Severity:    SeverityHigh,
					Description: "potentially dangerous dynamic code execution",
					File:        path,
					Line:        i + 1,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return findings, nil
}

// remoteCodeRule flags extensions that load code from remote URLs.
type remoteCodeRule struct{}

func (r *remoteCodeRule) ID() string { return "remote-code" }

func (r *remoteCodeRule) Scan(extDir string) ([]Finding, error) {
	var findings []Finding

	err := filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if strings.Contains(content, "http://") || strings.Contains(content, "https://") {
			// Check if it looks like a remote code load (not just a URL in
			// a match pattern).
			if strings.Contains(content, "fetch(") ||
				strings.Contains(content, "XMLHttpRequest") ||
				strings.Contains(content, "import(") {
				findings = append(findings, Finding{
					RuleID:      r.ID(),
					Severity:    SeverityMedium,
					Description: "extension loads code or data from remote URLs",
					File:        path,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return findings, nil
}

// unmarshalJSON is a minimal JSON unmarshal helper.
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
