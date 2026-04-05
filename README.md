# Ingo

Ingo is a Wails v2 + React desktop app for tracking AI usage across providers.

This first slice keeps things intentionally small:
- Wails v2 Go + React scaffold
- GitHub Copilot provider adapter at `internal/providers/copilot`
- Shared usage service + provider interface for future adapters
- Wails backend bindings via `App.GetUsage`, `App.GetCopilotUsage`, and `App.ListProviders`
- Simple CLI entry point at `cmd/ingo`

## Authentication

Ingo authenticates via the GitHub OAuth device flow — no environment variables or external CLI required.

On first launch, open the app and click **Sign in with GitHub**. You'll be shown a short code and a URL. Open the URL in your browser, enter the code, and approve. The app polls for approval and stores the resulting token at:

- **Linux:** `~/.config/ingo/credentials.json`
- **macOS:** `~/Library/Application Support/ingo/credentials.json`
- **Windows:** `%AppData%\ingo\credentials.json`

Credentials are shared between the GUI and CLI — once logged in via the app, the CLI works without any further setup.

If the token does not have the right access, or the account is not Copilot-enabled, GitHub will typically answer with `401`, `403`, or `404` and the adapter surfaces that response back to the caller.

## Run the CLI

The CLI shares credentials with the GUI. Log in once via the app, then:

```bash
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
