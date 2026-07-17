# Security Policy

## Supported versions

Security fixes are provided for the latest released version.

## Reporting a vulnerability

Please use GitHub's private vulnerability reporting for this repository. Do not include live OAuth tokens, account JSON files, or other credentials in a report.

If private reporting is unavailable, open a minimal issue asking for a private contact channel without disclosing the vulnerability or credentials.

## Credential handling

The plugin receives a Codex OAuth access token from CLIProxyAPI only for the authenticated model-catalog request. Credentials are kept in memory, are not persisted by the plugin, and are excluded from its logs and error messages.
