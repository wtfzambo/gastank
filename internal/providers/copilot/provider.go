package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gastank/internal/auth"
	"gastank/internal/usage"
)

const (
	ProviderName   = "github-copilot"
	defaultBaseURL = "https://api.github.com"
)

// TokenResolver provides an access token at fetch time.
type TokenResolver func(ctx context.Context) (string, error)

// Config wires dependencies for the provider.
type Config struct {
	HTTPClient    *http.Client
	BaseURL       string
	TokenResolver TokenResolver
	// CredStore, if non-nil, is used to resolve credentials via StoreTokenResolver.
	CredStore *auth.Store
}

// Provider implements usage.Provider for the GitHub Copilot internal user endpoint.
type Provider struct {
	httpClient    *http.Client
	baseURL       string
	tokenResolver TokenResolver
	credStore     *auth.Store // optional: clear on 401 to trigger re-auth
}

// quotaSnapshot represents the per-feature quota data returned by the API.
type quotaSnapshot struct {
	PercentRemaining *float64 `json:"percent_remaining"`
	Remaining        *float64 `json:"remaining"`
	QuotaRemaining   *float64 `json:"quota_remaining"`
	Unlimited        *bool    `json:"unlimited"`
	TimestampUTC     string   `json:"timestamp_utc"`
}

// quotaSnapshots groups all feature snapshots.
type quotaSnapshots struct {
	Chat                *quotaSnapshot `json:"chat"`
	Completions         *quotaSnapshot `json:"completions"`
	PremiumInteractions *quotaSnapshot `json:"premium_interactions"`
}

// apiResponse models the /copilot_internal/user response shape.
type apiResponse struct {
	CopilotPlan    string          `json:"copilot_plan"`
	QuotaResetDate string          `json:"quota_reset_date"`
	QuotaSnapshots *quotaSnapshots `json:"quota_snapshots"`
}

// NewProvider constructs a Provider with the given Config.
func NewProvider(cfg Config) *Provider {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	baseURL := cfg.BaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}

	resolver := cfg.TokenResolver
	if resolver == nil {
		if cfg.CredStore != nil {
			resolver = StoreTokenResolver(cfg.CredStore)
		} else {
			resolver = func(_ context.Context) (string, error) {
				return "", errors.New(
					"not authenticated: use the in-app login to connect your GitHub account",
				)
			}
		}
	}

	return &Provider{
		httpClient:    client,
		baseURL:       strings.TrimRight(baseURL, "/"),
		tokenResolver: resolver,
		credStore:     cfg.CredStore,
	}
}

// Name returns the canonical provider identifier.
func (p *Provider) Name() string {
	return ProviderName
}

// FetchUsage queries /copilot_internal/user and normalises the response.
func (p *Provider) FetchUsage(ctx context.Context) (*usage.UsageReport, error) {
	token, err := p.tokenResolver(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := p.baseURL + "/copilot_internal/user"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	// NOTE: These headers impersonate VS Code / GitHub Copilot Chat to access
	// the /copilot_internal/* endpoint. GitHub may gate this endpoint on specific
	// editor-version strings. If requests start returning 403 or 404 unexpectedly,
	// check whether GitHub has tightened the version gate and update these values.
	// The correct long-term fix is to use the officially documented API once one
	// is available for per-user quota data.
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Editor-Version", "vscode/1.96.2")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.26.7")
	req.Header.Set("X-Github-Api-Version", "2025-04-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Copilot usage: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Copilot usage response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			// Clear the stored credential so the next call forces re-auth.
			if p.credStore != nil {
				p.credStore.Clear(ProviderName)
			}
			return nil, fmt.Errorf("Copilot API returned %s (token invalid or revoked — log in again): %s",
				resp.Status, strings.TrimSpace(string(body)))
		}
		hint := ""
		if resp.StatusCode == http.StatusNotFound {
			hint = " (check that the account has Copilot access)"
		}
		return nil, fmt.Errorf("Copilot API returned %s%s: %s",
			resp.Status, hint, strings.TrimSpace(string(body)))
	}

	var payload apiResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode Copilot response: %w", err)
	}

	report := &usage.UsageReport{
		Provider:    p.Name(),
		RetrievedAt: time.Now().UTC().Format(time.RFC3339),
		Metrics:     make(map[string]float64),
		Metadata: map[string]string{
			"endpoint": "/copilot_internal/user",
		},
	}

	if payload.CopilotPlan != "" {
		report.Metadata["plan"] = payload.CopilotPlan
	}
	if payload.QuotaResetDate != "" {
		report.Metadata["quota_reset_date"] = payload.QuotaResetDate
	}

	if qs := payload.QuotaSnapshots; qs != nil {
		applySnapshot(report, "premium", qs.PremiumInteractions)
		applySnapshot(report, "chat", qs.Chat)
		applySnapshot(report, "completions", qs.Completions)
	}

	return report, nil
}

// applySnapshot writes a quota snapshot's fields into the report using a
// consistent key prefix, e.g. "premium_percent_remaining".
// If the quota is unlimited, a sentinel metric of 1 is written for
// "<prefix>_unlimited" and the percentage metrics are omitted.
func applySnapshot(report *usage.UsageReport, prefix string, snap *quotaSnapshot) {
	if snap == nil {
		return
	}
	if snap.Unlimited != nil && *snap.Unlimited {
		report.Metrics[prefix+"_unlimited"] = 1
		return
	}
	addMetricF(report.Metrics, prefix+"_percent_remaining", snap.PercentRemaining)
	addMetricF(report.Metrics, prefix+"_remaining", snap.Remaining)
	addMetricF(report.Metrics, prefix+"_quota_remaining", snap.QuotaRemaining)
}

// StoreTokenResolver returns a TokenResolver that reads from the credential
// store. If no valid credential is found, it returns a clear error directing
// the user to the in-app login.
func StoreTokenResolver(store *auth.Store) TokenResolver {
	return func(_ context.Context) (string, error) {
		if cred, ok := store.Get(ProviderName); ok && cred.Valid() {
			return cred.Token, nil
		}
		return "", errors.New(
			"not authenticated: use the in-app login to connect your GitHub account",
		)
	}
}

func addMetricF(metrics map[string]float64, key string, value *float64) {
	if value != nil {
		metrics[key] = *value
	}
}
