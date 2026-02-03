---
title: Security
layout: default
nav_order: 8
---

# Security

## Credential storage

The CLI stores your ClickUp API token in the operating system keyring:

- **macOS**: Keychain Access
- **Linux**: Secret Service API (GNOME Keyring, KDE Wallet)
- **Windows**: Windows Credential Manager

If the system keyring is unavailable, the CLI falls back to an encrypted file at `~/.config/clickup/auth.yml` with a warning.

Tokens are never stored in the config file (`config.yml`) or logged to stdout.

## Best practices

1. **Use the system keyring**: The default storage method. Avoid overriding it unless necessary.
2. **Scope your API token**: Use a personal API token with the minimum permissions your workflow needs.
3. **CI environments**: Pass tokens via environment variables and stdin, not as command-line arguments which may appear in process lists:

   ```sh
   echo "$CLICKUP_TOKEN" | clickup auth login --with-token
   ```

4. **Don't commit config files**: While `~/.config/clickup/config.yml` does not contain secrets, avoid committing it to shared repositories.
5. **Keep updated**: Always use the latest version for security patches.
6. **Logout when done**: Remove stored credentials with `clickup auth logout`.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Email security concerns to: **security@triptech.co.nz**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fixes (optional)

### Response timeline

| Step | Timeframe |
|------|-----------|
| Acknowledgment | Within 48 hours |
| Initial assessment | Within 5 business days |
| Resolution | Depends on severity, typically within 30 days |

We will credit reporters in release notes unless you prefer anonymity. We ask that you do not publicly disclose the issue until we have had time to address it.

## Supported versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |
