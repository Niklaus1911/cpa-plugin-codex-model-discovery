# Codex Model Discovery for CLIProxyAPI

`codex-model-discovery` is a [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) plugin that discovers the Codex models available to each OAuth account from OpenAI's authenticated model catalog.

It is useful when a newly available model is missing from CLIProxyAPI's built-in plan catalog. The plugin only supplies model metadata. Requests continue to use CLIProxyAPI's native Codex executor, authentication refresh, routing, exclusions, aliases, and model prefixes.

## Requirements

- CLIProxyAPI v7.2.81 or newer.
- A plugin-capable CLIProxyAPI build with CGO enabled.
- A Codex OAuth account containing an `access_token`.

Portable CGO-disabled CLIProxyAPI builds cannot load dynamic-library plugins.

## Install from the community registry

Add this source and enable plugins in `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Niklaus1911/cpa-plugin-codex-model-discovery/main/registry.json"
  configs:
    codex-model-discovery:
      enabled: true
```

Restart CLIProxyAPI, open the dashboard's Plugin Store, and install **Codex Model Discovery** from the community source. Restart once more after the first installation so the new dynamic library is loaded.

On Windows, use the official Windows amd64 CLIProxyAPI build and start it normally:

```powershell
.\CLIProxyAPI.exe --config .\config.yaml
```

`start-dev.cmd` is not required.

## Manual installation

Download the release archive matching the host, verify it against `checksums.txt`, and extract the library at the root of the configured plugin directory:

| Host | Archive | Library |
| --- | --- | --- |
| Windows x64 | `codex-model-discovery_0.1.0_windows_amd64.zip` | `codex-model-discovery.dll` |
| Linux x64 | `codex-model-discovery_0.1.0_linux_amd64.zip` | `codex-model-discovery.so` |
| Linux ARM64 | `codex-model-discovery_0.1.0_linux_arm64.zip` | `codex-model-discovery.so` |
| macOS Apple Silicon | `codex-model-discovery_0.1.0_darwin_arm64.zip` | `codex-model-discovery.dylib` |

Enable `plugins.enabled` and `plugins.configs.codex-model-discovery.enabled`, then restart the host.

## Configuration

All fields are optional:

```yaml
plugins:
  configs:
    codex-model-discovery:
      enabled: true
      base_url: "https://chatgpt.com/backend-api/codex"
      client_version: "0.144.1"
      user_agent: "codex_cli_rs/0.144.1 (Mac OS 26.3.1; arm64) iTerm.app/3.6.9"
      originator: "codex_cli_rs"
```

An account's `base_url` and `header:*` attributes override plugin defaults. CLIProxyAPI applies configured model exclusions, aliases, and prefixes after discovery.

## Failure behavior

- The first successful catalog is cached in memory for that account.
- Concurrent registrations for the same account share one upstream fetch.
- When a later fetch fails, the last successful catalog is returned.
- When no cache exists, the plugin declines the request so CLIProxyAPI uses its native plan-based catalog.
- The cache is cleared when discovery configuration changes or the process restarts.

Discovery is synchronous while an account is registered. The current plugin API does not let a plugin return a fallback immediately and asynchronously request model re-registration later.

The host HTTP callback applies CLIProxyAPI's global proxy configuration. It does not currently carry an account-specific `proxy_url` or a custom `Host` override into dynamic plugins.

## Security

CLIProxyAPI plugins are trusted in-process native code. This plugin receives the OAuth access token needed for discovery. It does not persist credentials and never writes tokens, raw account JSON, authorization headers, or upstream error bodies to logs.

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## Development

Go 1.26+ and a C compiler are required:

```bash
go test ./...
go test -race ./...
go vet ./...
CGO_ENABLED=1 go build -trimpath -buildmode=c-shared -o build/codex-model-discovery.so .
```

Release artifacts are built on native GitHub-hosted Windows, Linux, and macOS runners.
