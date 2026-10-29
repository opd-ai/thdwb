# Security Policy

## Supported Versions

We release security updates for the following versions:

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅        |
| < 0.1   | ❌        |

## Reporting a Vulnerability

We take the security of thdwb seriously. If you believe you have found a
security vulnerability, please report it to us responsibly.

### How to Report

Please email security@thdwb.dev with:

1. A clear description of the vulnerability
2. Steps to reproduce the issue
3. The affected component(s) and version(s)
4. Any potential impact
5. Suggested remediation (if known)

Please do not file public GitHub issues for security vulnerabilities.

### What to Expect

- You will receive an acknowledgment within 5 business days.
- We will provide an initial assessment within 10 business days.
- We will keep you informed of our progress throughout the remediation process.
- We will credit you in our security advisory (unless you prefer to remain anonymous).

### Disclosure Policy

We follow a coordinated disclosure process:

1. We confirm receipt of your report.
2. We investigate and develop a fix.
3. We prepare a security advisory describing the vulnerability and its fix.
4. We release the fix and advisory, typically within 90 days of the initial report.
5. We credit the reporter (if desired).

## Security Best Practices for Users

- Always download thdwb from official sources.
- Verify checksums of downloaded packages.
- Keep your browser and extensions up to date.
- Only install extensions from trusted sources.
- Review extension permissions before installing.

## Security Features

thdwb includes the following security features:

- Same-Origin Policy (SOP) enforcement
- Cross-Origin Resource Sharing (CORS) validation
- Content Security Policy (CSP) enforcement
- Per-extension sandboxed JavaScript VMs
- Container tab isolation (separate cookie jars and storage)
- Certificate validation and HSTS support
- Secure extension package checksum verification
