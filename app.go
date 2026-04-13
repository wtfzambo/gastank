package main

import (
	"context"
	"errors"
	"log"

	"gastank/internal/auth"
	githubauth "gastank/internal/auth/github"
	"gastank/internal/providers/copilot"
	"gastank/internal/usage"

	"github.com/wailsapp/wails/v3/pkg/application"
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

// App is the Wails v3 service. It exposes auth and usage methods to the
// frontend and implements ServiceStartup so it receives the app context.
type App struct {
	ctx          context.Context
	credStore    *auth.Store
	credsPath    string // path to the on-disk credentials file
	usageService *usage.Service
	deviceFlow   *githubauth.DeviceFlow
}

// NewApp creates a new App service instance, loading credentials from disk.
func NewApp() *App {
	store := auth.NewStore()

	credsPath, err := auth.DefaultCredentialsPath()
	if err != nil {
		log.Printf("gastank: could not resolve credentials path: %v", err)
	} else {
		if err := store.Load(credsPath); err != nil {
			log.Printf("gastank: could not load credentials: %v", err)
		}
	}

	return &App{
		credStore: store,
		credsPath: credsPath,
		usageService: usage.NewService(
			copilot.NewProvider(copilot.Config{CredStore: store}),
		),
		deviceFlow: githubauth.NewDeviceFlow(nil),
	}
}

// ServiceStartup implements application.ServiceStartup.
// Called by Wails v3 during app startup; stores the context for use in
// background operations (device-flow polling, usage fetches).
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	return nil
}

// save persists the credential store to disk. Failures are logged but not
// returned to the caller — a save error should never break the in-memory flow.
func (a *App) save() {
	if a.credsPath == "" {
		return
	}
	if err := a.credStore.Save(a.credsPath); err != nil {
		log.Printf("gastank: could not save credentials: %v", err)
	}
}

// --- Auth methods (Wails-callable) ---

// GetAuthStatus returns whether the Copilot provider has a valid credential.
func (a *App) GetAuthStatus() AuthStatus {
	if cred, ok := a.credStore.Get(copilot.ProviderName); ok && cred.Valid() {
		return AuthStatus{Authenticated: true, Source: string(cred.Source)}
	}
	return AuthStatus{Authenticated: false}
}

// StartGitHubLogin begins the OAuth device flow and returns the user-facing
// code and URL. The caller should display these to the user, then poll with
// PollGitHubLogin.
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
// has approved (credential is stored and persisted), false when still pending,
// and an error on fatal states (expired, denied, network error).
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
	a.save()
	return true, nil
}

// LogOut clears the stored credential for the Copilot provider and persists.
func (a *App) LogOut() {
	a.credStore.Clear(copilot.ProviderName)
	a.save()
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

// GetVersion returns the build version for the current app binary.
func (a *App) GetVersion() string {
	return Version
}

// ListProviders returns the provider adapters currently wired into the app.
func (a *App) ListProviders() []string {
	return a.usageService.Providers()
}
