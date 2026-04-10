# Gastank — AI Consumption Tracking Tray App

## What

A cross-platform system tray application that displays AI provider consumption/usage (GitHub Copilot, Cloud Code, OpenAI, Anthropic, etc.) in a click-to-toggle popover with charts and cost breakdowns.

## Why

AI tool subscriptions are becoming expensive and opaque. No single lightweight, always-on dashboard exists to show usage across providers. Users either check each provider separately or guess.

## Who

Developers and power users with multiple AI subscriptions who want visibility into usage and spend.

## Tech Stack

- **Backend:** Go (API polling, provider adapters)
- **Frontend:** Wails (Go + Webview) with React for the popover UI
- **Platforms:** macOS + Windows (cross-platform native binaries)
- **UX:** Click-to-toggle popover (not hover — more reliable cross-OS)

## Development Principles

1. **Documentation is a must** — but wrong documentation is worse than no documentation
2. **Skateboard before Lamborghini** — build incrementally, MVP first. Ship the thing that gets from A to B the quickest and easiest. Improve and expand later if needed
3. **YAGNI (Less is more)** — don't build what you don't need yet
4. **Locality of behavior** — things that work together go together
5. **Test-driven development** — but stay reasonable. Don't test for the sake of testing
6. **Best practices without zealotry** — clean code matters, but shipping matters more

## Pipeline

Donna → Lobster → Felicity → Pi (via acpx) → OpenSpec → GitHub repo

- OpenSpec for spec-driven development
- Feature branches + PRs
- Donna reviews before merge

## Repo

- GitHub: `donnaknows/gastank` (private)
- Local: `~/mystuff/projects/gastank/`
