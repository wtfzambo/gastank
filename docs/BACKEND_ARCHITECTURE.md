# Gastank Backend Architecture

## Overview

Gastank is a system-tray desktop app (+ optional CLI) that monitors AI provider usage quotas. The backend is written in Go and uses [Wails v3](https://v3.wails.io/) to bridge a React frontend with native Go services. There is no server, no database, no network listener — it's a local-only desktop app.

## Project Layout

```
gastank/
├── main.go                     # Wails app entry point (tray window + system tray)
├── app.go                      # Wails service: auth + usage methods exposed to frontend
├── version.go                  # Build-time version variable
├── tray.go                     # Embedded tray icon (PNG → []byte)
├── cmd/gastank/main.go         # Standalone CLI binary (no GUI)
├── internal/
│   ├── auth/
│   │   ├── credential.go       # Credential type + thread-safe Store (load/save JSON)
│   │   └── github/
│   │       └── deviceflow.go   # GitHub OAuth Device Flow implementation
│   ├── usage/
│   │   ├── provider.go         # Provider interface + UsageReport type
│   │   └── service.go          # Provider registry (fetch by name)
│   └── providers/
│       └── copilot/
│           └── provider.go     # GitHub Copilot usage fetcher
├── frontend/                   # React + Vite (compiled into Go binary via embed)
├── build/                      # Platform-specific Taskfiles + packaging configs
├── scripts/                    # Release helper scripts
└── .github/workflows/          # CI + Release workflows
```

## How It Works

### 1. Application Lifecycle (`main.go`)

The app starts as a **system tray application** — no dock icon, no taskbar entry:

- Creates a frameless, always-on-top 360x420 window (hidden by default)
- Attaches the window to a system tray icon (left-click toggles, right-click shows menu)
- On macOS, uses `ActivationPolicyAccessory` to stay out of the Dock / Cmd+Tab
- Window closing hides instead of quitting (tray app convention)
- The React frontend is embedded into the binary via `//go:embed all:frontend/dist`

### 2. Wails Service Layer (`app.go`)

`App` is registered as a Wails v3 service. Every public method on `App` is automatically callable from the frontend via Wails bindings (generated JS in `frontend/bindings/`).

**Exposed methods:**

| Method | What it does |
|---|---|
| `GetAuthStatus()` | Checks if a valid Copilot credential exists |
| `StartGitHubLogin()` | Begins OAuth device flow, returns user code + URL |
| `PollGitHubLogin(code)` | Polls token endpoint once — returns true when approved |
| `LogOut()` | Clears stored credential |
| `GetCopilotUsage()` | Fetches current quota data from GitHub |
| `GetUsage(provider)` | Generic version — fetches by provider name |
| `GetVersion()` | Returns build version string |
| `ListProviders()` | Returns registered provider names |

### 3. Authentication (`internal/auth/`)

**Credential Store** (`credential.go`):
- Thread-safe in-memory map of provider-key → `Credential`
- Persists to `<UserConfigDir>/gastank/credentials.json`
- Atomic writes (write to `.tmp`, then rename)
- Loads on startup, saves after login/logout
- Skips expired or empty credentials on load

**GitHub Device Flow** (`github/deviceflow.go`):
- Uses GitHub's OAuth device flow (no redirect URI needed)
- Borrows the VS Code Copilot Chat OAuth client ID (`Iv1.b507a08c87ecfe98`)
- Frontend shows the user code + verification URL, then polls `PollGitHubLogin()` on a timer
- Returns sentinel errors for transient states (`ErrAuthorizationPending`, `ErrSlowDown`) and terminal states (`ErrExpired`, `ErrAccessDenied`)

### 4. Usage Providers (`internal/usage/` + `internal/providers/`)

**Architecture**: simple provider registry pattern.

```
Provider interface {
    Name() string
    FetchUsage(ctx) (*UsageReport, error)
}
```

`Service` holds a `map[string]Provider`. `Fetch(ctx, name)` dispatches to the right one. Currently only one provider exists.

**Copilot Provider** (`providers/copilot/provider.go`):
- Hits `GET https://api.github.com/copilot_internal/user` (undocumented endpoint)
- Impersonates VS Code via request headers (required to access this endpoint)
- Parses `quota_snapshots` for three quota categories: `premium_interactions`, `chat`, `completions`
- Each category reports: `percent_remaining`, `remaining`, `quota_remaining`, or `unlimited`
- On 401/403: clears the credential store to force re-auth
- On 404: hints that the account may lack Copilot access

**UsageReport** shape:
```json
{
  "provider": "github-copilot",
  "retrievedAt": "2025-04-13T10:00:00Z",
  "metrics": {
    "premium_percent_remaining": 85.5,
    "chat_percent_remaining": 100,
    "completions_unlimited": 1
  },
  "metadata": {
    "plan": "business",
    "quota_reset_date": "2025-05-01",
    "endpoint": "/copilot_internal/user"
  }
}
```

### 5. CLI Binary (`cmd/gastank/main.go`)

Separate binary, no GUI dependency. Shares the same `internal/` packages.

```bash
gastank usage                  # fetch copilot usage, print JSON
gastank usage github-copilot   # explicit provider name
gastank --version              # print version
```

Uses the same credential store file as the tray app — login once via the GUI, then the CLI can read the saved token.

### 6. Version Injection

- `version.go` declares `var Version = "dev"` (root package, GUI binary)
- `cmd/gastank/main.go` declares `var version = "dev"` (CLI binary)
- Both are overridden at build time via `-ldflags "-X main.Version=<tag>"`
- Platform Taskfiles include this flag when `VERSION` var is set
- `scripts/set-release-version.sh` separately stamps `build/config.yml` and `build/linux/nfpm/nfpm.yaml` (for NSIS/nfpm package metadata)

## Build & Release

### Local Development

```bash
wails3 dev          # hot-reload dev mode
go test ./...       # run tests
go run . --version  # check version
```

### CI (`ci.yml`)

Runs on push/PR to main: `go test ./...` + `npm run build` (frontend).

### Release (`release.yml`)

Triggered by pushing a `v*` tag. Builds on all three platforms in parallel, then publishes a GitHub Release with auto-generated release notes.

**Artifacts produced:**
- macOS: `gastank-macos-universal.zip` (universal arm64+amd64 .app bundle)
- Windows: `gastank.exe` + `gastank-windows-amd64-installer.exe` (NSIS installer)
- Linux: `gastank-linux-amd64` binary + AppImage + .deb + .rpm + .pkg.tar.zst

### Install Script

`scripts/install.sh` — curl-pipe installer for macOS (copies .app to /Applications) and Linux (downloads AppImage to `~/.local/bin/`).

## Known Limitations

1. **Undocumented API**: The Copilot usage endpoint is not part of GitHub's public API. It could change or be restricted at any time.
2. **Borrowed OAuth client ID**: Uses VS Code's client ID. A dedicated OAuth app registration would be more robust.
3. **No auto-update**: Users must manually re-run the install script or download new releases.
4. **CLI binary not distributed**: The release workflow only builds the GUI binary. The CLI at `cmd/gastank/` is for local development use.
