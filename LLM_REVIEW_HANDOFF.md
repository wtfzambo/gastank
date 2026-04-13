# LLM Review Handoff

## Scope

This handoff covers work completed for GitHub issues `#6`, `#10`, and `#8` in this order:

1. `#6` Auto-refresh consumption display every 5-10 minutes
2. `#10` Adopt TypeUI as frontend design system
3. `#8` Set up CI/CD, distribution, and auto-update pipeline

The goal of this document is to help another LLM review the implementation quickly and focus on the risky parts.

## High-Level Summary

### Issue #6

Implemented automatic polling of Copilot usage in the React tray UI.

- Added a 5-minute auto-refresh interval in `frontend/src/App.tsx`
- Prevented overlapping refresh requests with a ref guard
- Preserved the last successful snapshot if a background refresh fails
- Exposed refresh state in the UI with `Live`, `Refreshing`, and `Stale` states
- Kept manual refresh support

### Issue #10

Implemented a compact native-dark tray UI refresh and added a lightweight design-system spec.

- Reworked the tray UI in `frontend/src/App.tsx`
- Replaced the previous ad-hoc CSS with a more cohesive popover treatment in `frontend/src/App.css`
- Removed the old Wails scaffold stylesheet `frontend/src/style.css`
- Added `.agents/skills/design-system/SKILL.md` as a local design-system/spec file for future agent work

Important note: during repo exploration, `TypeUI` looked more like a design/spec workflow than a runtime React component library for this codebase, so the implementation treated issue `#10` as "polished UI + local design-system spec" rather than adding a runtime UI package.

### Issue #8

Implemented release and distribution scaffolding for all platforms using GitHub Actions and existing Wails v3 packaging tasks.

- Added CI workflow: `.github/workflows/ci.yml`
- Added release workflow: `.github/workflows/release.yml`
- Added release helper scripts:
  - `scripts/install.sh`
  - `scripts/set-release-version.sh`
- Added app version plumbing:
  - `version.go`
  - `App.GetVersion()` in `app.go`
  - `--version` support in `main.go`
  - `version`/`--version` support in `cmd/gastank/main.go`
  - regenerated Wails bindings in `frontend/bindings/gastank/app.js`
- Fixed packaging/taskfile issues discovered while implementing:
  - version ldflags for darwin/windows/linux taskfiles
  - Windows installer task path/directory mismatch
  - missing Linux package metadata via `build/linux/nfpm/nfpm.yaml`
- Updated `README.md` with install guidance

Important note: signing, notarization, and real auto-update integration were intentionally not implemented. The user explicitly wanted simple GitHub Releases + install script distribution and to leave those harder platform-security/update concerns as follow-up work.

## Files Added

- `LLM_REVIEW_HANDOFF.md`
- `version.go`
- `.agents/skills/design-system/SKILL.md`
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `build/linux/nfpm/nfpm.yaml`
- `scripts/install.sh`
- `scripts/set-release-version.sh`

## Files Changed

- `README.md`
- `app.go`
- `build/config.yml`
- `build/darwin/Taskfile.yml`
- `build/linux/Taskfile.yml`
- `build/windows/Taskfile.yml`
- `build/windows/installer/project.nsi`
- `cmd/gastank/main.go`
- `frontend/bindings/gastank/app.js`
- `frontend/src/App.css`
- `frontend/src/App.tsx`
- `frontend/src/main.tsx`
- `main.go`

## Files Removed

- `frontend/src/style.css`

## What Was Verified Locally

The following commands were run successfully after the implementation:

```bash
cd frontend && npm run build
go test ./...
go run . --version
go run ./cmd/gastank --version
bash -n scripts/install.sh
bash -n scripts/set-release-version.sh
```

## What Was Not Fully Proven Locally

These parts were not fully end-to-end proven in this session:

- The GitHub Actions release workflow has not been exercised by pushing a real tag
- Cross-platform packaging outputs have not all been built locally on their native CI runners
- The installer script depends on actual published release artifacts existing on GitHub Releases
- Visual review of the tray UI in the live app was not documented here beyond successful frontend build

## Suggested Review Focus

### Highest Priority

1. Review `.github/workflows/release.yml`
   - Check whether each platform runner has the right dependencies
   - Check whether the expected release artifact names match what Wails actually emits
   - Check whether `wails3 package` plus the current taskfiles is sufficient on macOS, Windows, and Linux

2. Review packaging/taskfile assumptions
   - `build/darwin/Taskfile.yml`
   - `build/windows/Taskfile.yml`
   - `build/linux/Taskfile.yml`
   - `build/windows/installer/project.nsi`
   - `build/linux/nfpm/nfpm.yaml`

3. Review the install script behavior
   - `scripts/install.sh`
   - Confirm that it targets the intended desktop release assets
   - Confirm the Linux AppImage resolution approach is robust enough for GitHub release HTML

### Medium Priority

4. Review the frontend polling behavior in `frontend/src/App.tsx`
   - Ensure auto-refresh interval cleanup is correct
   - Ensure auth failures still route the user to login cleanly
   - Ensure stale data UX is appropriate for a tray app

5. Review the UI/layout changes
   - `frontend/src/App.tsx`
   - `frontend/src/App.css`
   - Confirm the compact dark tray layout is appropriate on desktop-sized popovers
   - Confirm no accessibility or state regressions were introduced

### Lower Priority

6. Review version plumbing
   - `version.go`
   - `main.go`
   - `cmd/gastank/main.go`
   - `app.go`
   - `frontend/bindings/gastank/app.js`
   - Confirm runtime/build-time version handling is internally consistent

## Known Caveats / Risk Notes

1. `TypeUI` interpretation
   The issue said to integrate TypeUI as a design system. In this repo, that did not map cleanly to a runtime component library, so the implementation used a local design-system/spec file plus direct UI polish. Review whether that interpretation matches project intent.

2. Release workflow correctness is the riskiest area
   The repository originally had incomplete or inconsistent packaging scaffolding. Some of that was fixed during this pass, but the GitHub Actions release path is still the least proven part because it needs real CI execution.

3. Linux installer strategy
   The install script currently resolves an AppImage from the GitHub release page and installs it as `~/.local/bin/gastank-app`. Review whether this is the desired UX, or whether a different Linux install flow should be preferred.

4. `build/config.yml` and `build/linux/nfpm/nfpm.yaml` are version-stamped by script
   The repo keeps default versions checked in. Release builds are expected to run `scripts/set-release-version.sh` before packaging.

5. No real updater service was added
   Although issue `#8` mentioned auto-update, this was intentionally left as follow-up work per user direction.

## Questions the Reviewer Should Answer

1. Does `release.yml` look correct for Wails v3 packaging on macOS, Windows, and Linux?
2. Are the packaging taskfile changes minimal and correct, or did they introduce any hidden breakage?
3. Is the `install.sh` approach appropriate for this repo's intended distribution model?
4. Is the `#10` interpretation acceptable, or should the issue be revisited with a stricter TypeUI/runtime integration requirement?
5. Are there any frontend regressions or UX concerns in the new tray layout or auto-refresh behavior?

## Short Reviewer Prompt

If you want to hand this to another LLM directly, use something like:

```text
Review the changes in this repo for issues #6, #10, and #8. Focus primarily on bugs, regressions, packaging/release risks, and mismatches between the implementation and the intended issue scope. Pay special attention to .github/workflows/release.yml, the Wails taskfile changes, scripts/install.sh, and the frontend polling logic in frontend/src/App.tsx. Ignore style nitpicks unless they hide a real risk.
```
