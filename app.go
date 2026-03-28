package main

import (
	"context"
	"fmt"

	"ingo/internal/providers/copilot"
	"ingo/internal/usage"
)

// App struct
type App struct {
	ctx          context.Context
	usageService *usage.Service
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		usageService: usage.NewService(
			copilot.NewProvider(copilot.Config{}),
		),
	}
}

// startup is called when the app starts. The context is saved so we can call runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, it's show time.", name)
}

// GetUsage fetches usage data for a named provider.
func (a *App) GetUsage(providerName string) (*usage.UsageReport, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	return a.usageService.Fetch(ctx, providerName)
}

// GetCopilotUsage fetches GitHub Copilot usage data through the provider adapter.
func (a *App) GetCopilotUsage() (*usage.UsageReport, error) {
	return a.GetUsage(copilot.ProviderName)
}

// ListProviders returns the provider adapters currently wired into the app.
func (a *App) ListProviders() []string {
	return a.usageService.Providers()
}
