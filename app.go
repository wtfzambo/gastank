package main

import (
	"context"
	"errors"

	"ingo/internal/auth"
	githubauth "ingo/internal/auth/github"
	"ingo/internal/providers/copilot"
	"ingo/internal/usage"
)

// DeviceFlowState is returned to the frontend when starting GitHub login.
type DeviceFlowState struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationURI"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

// AuthStatus is returned to the frontend to describe the current auth state
// for a provider.
type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	Source        string `json:"source,omitempty"`
}

// App struct
type App struct {
	ctx          context.Context
	credStore    *auth.Store
	usageService *usage.Service
	deviceFlow   *githubauth.DeviceFlow
}

// NewApp creates a new App application struct.
func NewApp() *App {
	store := auth.NewStore()
	return &App{
		credStore: store,
		usageService: usage.NewService(
			copilot.NewProvider(copilot.Config{CredStore: store}),
		),
		deviceFlow: githubauth.NewDeviceFlow(nil),
	}
}

// startup is called when the app starts. The context is saved
// so we can call runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// --- Auth methods (Wails-callable) ---

// GetAuthStatus returns whether the Copilot provider has a valid credential.
func (a *App) GetAuthStatus() AuthStatus {
	cred, ok := a.credStore.Get(copilot.ProviderName)
	if ok && cred.Valid() {
		return AuthStatus{Authenticated: true, Source: string(cred.Source)}
	}
	// Also accept env-var tokens as "authenticated" so the UI doesn't
	// prompt for device flow when a token is already available via env.
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := copilot.EnvTokenResolver(ctx); err == nil {
		return AuthStatus{Authenticated: true, Source: string(auth.SourceEnvVar)}
	}
	return AuthStatus{Authenticated: false}
}

// StartGitHubLogin begins the OAuth device flow and returns the user-facing
// code and URL. The caller should display these to the user, then call
// PollGitHubLogin with the returned DeviceCode.
func (a *App) StartGitHubLogin() (*DeviceFlowState, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := a.deviceFlow.Start(ctx)
	if err != nil {
		return nil, err
	}
	return &DeviceFlowState{
		DeviceCode:      resp.DeviceCode,
		UserCode:        resp.UserCode,
		VerificationURI: resp.VerificationURI,
		ExpiresIn:       resp.ExpiresIn,
		Interval:        resp.Interval,
	}, nil
}

// PollGitHubLogin polls the token endpoint once. Returns true when the user
// has approved, false when still pending (ErrAuthorizationPending /
// ErrSlowDown), and an error on fatal states (expired, denied, network error).
// The frontend is responsible for the polling interval.
func (a *App) PollGitHubLogin(deviceCode string) (bool, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	cred, err := a.deviceFlow.Poll(ctx, deviceCode)
	if err != nil {
		if errors.Is(err, githubauth.ErrAuthorizationPending) ||
			errors.Is(err, githubauth.ErrSlowDown) {
			return false, nil
		}
		return false, err
	}
	a.credStore.Set(copilot.ProviderName, cred)
	return true, nil
}

// LogOut clears the stored credential for the Copilot provider.
func (a *App) LogOut() {
	a.credStore.Clear(copilot.ProviderName)
}

// --- Usage methods (Wails-callable) ---

// GetUsage fetches usage data for a named provider.
func (a *App) GetUsage(providerName string) (*usage.UsageReport, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.usageService.Fetch(ctx, providerName)
}

// GetCopilotUsage fetches GitHub Copilot usage through the provider adapter.
func (a *App) GetCopilotUsage() (*usage.UsageReport, error) {
	return a.GetUsage(copilot.ProviderName)
}

// ListProviders returns the provider adapters currently wired into the app.
func (a *App) ListProviders() []string {
	return a.usageService.Providers()
}
