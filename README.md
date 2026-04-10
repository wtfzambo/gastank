# Gastank

Gastank is a cross-platform desktop tray app for tracking AI token usage across providers, built with Wails v3 + React.

This first slice keeps things intentionally small:
- Wails v3 Go + React scaffold with native system tray
- GitHub Copilot provider adapter at `internal/providers/copilot`
- Shared usage service + provider interface for future adapters
- Wails backend bindings via `App.GetUsage`, `App.GetCopilotUsage`, and `App.ListProviders`
- Simple CLI entry point at `cmd/gastank`

## Authentication

Gastank authenticates via the GitHub OAuth device flow — no environment variables or external CLI required.

On first launch, open the app and click **Sign in with GitHub**. You'll be shown a short code and a URL. Open the URL in your browser, enter the code, and approve. The app polls for approval and stores the resulting token at:

- **Linux:** `~/.config/gastank/credentials.json`
- **macOS:** `~/Library/Application Support/gastank/credentials.json`
- **Windows:** `%AppData%\gastank\credentials.json`

Credentials are shared between the GUI and CLI — once logged in via the app, the CLI works without any further setup.

If the token does not have the right access, or the account is not Copilot-enabled, GitHub will typically answer with `401`, `403`, or `404` and the adapter surfaces that response back to the caller.

## Run the CLI

The CLI shares credentials with the GUI. Log in once via the app, then:

```bash
go run ./cmd/gastank usage github-copilot
```

## Run tests

```bash
go test ./...
```

## Live development

```bash
task dev
```

## Build

```bash
task build
```

## Package (macOS .app)

```bash
task package
open bin/gastank.app
```
