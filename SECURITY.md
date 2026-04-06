# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in go-routeros, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email: **security@cepatkilat.tech**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge your report within 48 hours and aim to release a fix within 7 days for critical issues.

## Security Considerations

This library handles credentials and communicates with network devices. Users should:

- **Never hardcode credentials** in source code. Use environment variables or secret managers.
- **Use TLS** when possible (`rest.WithInsecureSkipVerify` and `api.WithTLS` / `api.WithTLSConfig`).
- **Restrict network access** to RouterOS management interfaces.
- **Keep RouterOS firmware updated** to patch known vulnerabilities (e.g., CVE-2025-10948).
