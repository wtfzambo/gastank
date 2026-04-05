# Ingo

Ingo is a Wails v2 + React desktop app for tracking AI usage across providers.

This first slice keeps things intentionally small:
- Wails v2 Go + React scaffold
- GitHub Copilot provider adapter at `internal/providers/copilot`
- Shared usage service + provider interface for future adapters
- Wails backend bindings via `App.GetUsage`, `App.GetCopilotUsage`, and `App.ListProviders`
- Simple CLI entry point at `cmd/ingo`

## GitHub Copilot adapter

The Copilot adapter calls `GET https://api.github.com/copilot_internal/user` and normalizes the response into a shared `UsageReport` shape.

Auth resolution order:
1. Saved credentials (stored by in-app device-flow login)
2. `GITHUB_TOKEN` env var
3. `GH_TOKEN` env var

For interactive use, the app exposes a GitHub device-flow login that stores the resulting credential at `<os.UserConfigDir>/ingo/credentials.json`.

If the token does not have the right access, or the account is not Copilot-enabled, GitHub will typically answer with `401`, `403`, or `404` and the adapter surfaces that response back to the caller.

## Run the CLI

```bash
export GITHUB_TOKEN=YOUR_GITHUB_TOKEN
# or: GH_TOKEN is also accepted

go run ./cmd/ingo usage github-copilot
```

## Run tests

```bash
go test ./...
```

## Live development

```bash
wails dev
```

## Build

Ubuntu 24.04 ships `webkit2gtk-4.1`, while Wails v2.12.0 still asks `pkg-config` for `webkit2gtk-4.0`. This repo includes a tiny local shim under `build/linux/pkgconfig/` so the build still works without touching the system install.

```bash
PKG_CONFIG_PATH=$PWD/build/linux/pkgconfig:${PKG_CONFIG_PATH} wails build
```
