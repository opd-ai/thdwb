# thdwb Security Advisory Process

This document describes the process for handling security vulnerabilities in
thdwb, from initial report to public disclosure.

## Overview

thdwb follows a coordinated vulnerability disclosure process. Security
vulnerabilities are handled privately until a fix is available, at which point
a security advisory is published.

## Reporting

### How to Report

Email security@thdwb.dev with:

1. A clear description of the vulnerability
2. Steps to reproduce the issue
3. The affected component(s) and version(s)
4. Any potential impact
5. Suggested remediation (if known)

**Do not** file public GitHub issues for security vulnerabilities.

### What to Expect

| Step | Timeline | Description |
|------|----------|-------------|
| Acknowledgment | 5 business days | We confirm receipt of your report |
| Initial assessment | 10 business days | We evaluate validity and severity |
| Status updates | Every 2 weeks | We keep you informed of progress |
| Fix release | Target 90 days | We release a fix and advisory |

## Triage

### Severity Classification

We use the following severity levels:

| Severity | Description | Example |
|----------|-------------|---------|
| 🔴 Critical | Remote code execution, full browser compromise | Sandbox escape, arbitrary code execution |
| 🟠 High | Significant data exposure, privilege escalation | Cross-container data leak, SOP bypass |
| 🟡 Medium | Limited data exposure, partial bypass | CSP bypass, cookie jar leak |
| ⚪ Low | Minor information disclosure, theoretical attack | Version info disclosure, non-sensitive data leak |

### Triage Steps

1. **Receive**: Security report is received at security@thdwb.dev
2. **Acknowledge**: Confirm receipt within 5 business days
3. **Assess**: Validate the vulnerability and assign severity
4. **Assign**: Assign to a maintainer for fix development
5. **Fix**: Develop, test, and review the fix
6. **Release**: Publish the fix and security advisory
7. **Credit**: Acknowledge the reporter (unless they prefer anonymity)

## Fix Development

### Branch

Fixes are developed on a private branch (not pushed to the public repository)
until ready for release.

### Testing

All security fixes must include:

- A regression test demonstrating the vulnerability is fixed
- Tests for related code paths
- `go test -race ./...` passing
- `go vet ./...` passing

### Review

Security fixes require review by at least two maintainers before release.

## Release

### Patch Release

Security fixes are released as patch versions (e.g., 0.1.1 from 0.1.0).

### Security Advisory

Each security fix includes a published advisory with:

- CVE identifier (if assigned)
- Affected versions
- Fixed version
- Severity rating
- Description of the vulnerability
- Steps to reproduce (if appropriate)
- Remediation instructions
- Credit to the reporter

### Advisory Template

See [SECURITY_ADVISORY_TEMPLATE.md](SECURITY_ADVISORY_TEMPLATE.md) for the
template used for security advisories.

## Credit

Reporters are credited in security advisories unless they request anonymity.
Credit includes the reporter's name or handle and a link to their profile (if
provided).

## Contact

- Security reports: security@thdwb.dev
- PGP key: Available on request
