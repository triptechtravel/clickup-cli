# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email security concerns to: **info@campermate.com**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (optional)

### What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 5 business days
- **Resolution Timeline**: Depends on severity, typically within 30 days

### Disclosure Policy

- We will work with you to understand and resolve the issue
- We will credit reporters in release notes (unless you prefer anonymity)
- We ask that you do not publicly disclose until we've had time to address the issue

## Security Best Practices

When using this tool:

1. **Use the system keyring**: The CLI stores tokens in the OS keyring by default (macOS Keychain, etc.). Avoid storing tokens in plain text.
2. **Keep your API token scoped**: Use a personal API token with the minimum required permissions.
3. **Don't commit config files**: The CLI config at `~/.config/clickup/config.yml` does not contain secrets, but avoid committing it to shared repos.
4. **Keep updated**: Always use the latest version for security patches.
5. **CI tokens**: In CI environments, pass tokens via environment variables and stdin (`echo "$CLICKUP_TOKEN" | clickup auth login --with-token`), not as command-line arguments.
