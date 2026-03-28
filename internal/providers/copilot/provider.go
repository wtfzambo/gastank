package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ingo/internal/usage"
)

const (
	ProviderName   = "github-copilot"
	defaultBaseURL = "https://api.github.com"
	apiVersion     = "2022-11-28"
)

// TokenResolver provides an access token at fetch time.
type TokenResolver func(ctx context.Context) (string, error)

// Config wires dependencies for the provider.
type Config struct {
	HTTPClient    *http.Client
	BaseURL       string
	TokenResolver TokenResolver
}

// Provider implements usage.Provider for the GitHub Copilot usage endpoint.
type Provider struct {
	httpClient    *http.Client
	baseURL       string
	tokenResolver TokenResolver
}

type apiResponse struct {
	CopilotPlan              string   `json:"copilot_plan"`
	PeriodStart              string   `json:"period_start"`
	PeriodEnd                string   `json:"period_end"`
	TotalPremiumRequests     *float64 `json:"total_premium_requests"`
	PremiumRequestsLimit     *float64 `json:"premium_requests_limit"`
	RemainingPremiumRequests *float64 `json:"remaining_premium_requests"`
	TotalChatTurns           *float64 `json:"total_chat_turns"`
	TotalCompletions         *float64 `json:"total_completions"`
	TotalAcceptanceCount     *float64 `json:"total_acceptance_count"`
}

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
		resolver = EnvTokenResolver
	}

	return &Provider{
		httpClient:    client,
		baseURL:       strings.TrimRight(baseURL, "/"),
		tokenResolver: resolver,
	}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) FetchUsage(ctx context.Context) (*usage.UsageReport, error) {
	token, err := p.tokenResolver(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/user/copilot/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", "ingo/0.1")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub Copilot usage: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read GitHub Copilot usage response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		hint := ""
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			hint = " (check that the token has the copilot scope and that the endpoint is enabled for this account)"
		}

		return nil, fmt.Errorf("GitHub Copilot usage API returned %s%s: %s", resp.Status, hint, strings.TrimSpace(string(body)))
	}

	var payload apiResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode GitHub Copilot usage response: %w", err)
	}

	report := &usage.UsageReport{
		Provider:    p.Name(),
		RetrievedAt: time.Now().UTC().Format(time.RFC3339),
		Metrics:     make(map[string]float64),
		Metadata: map[string]string{
			"endpoint": "/user/copilot/usage",
		},
	}

	if payload.CopilotPlan != "" {
		report.Metadata["plan"] = payload.CopilotPlan
	}

	if payload.PeriodStart != "" {
		periodStart, err := time.Parse(time.RFC3339, payload.PeriodStart)
		if err != nil {
			return nil, fmt.Errorf("parse period_start: %w", err)
		}
		report.PeriodStart = periodStart.UTC().Format(time.RFC3339)
	}

	if payload.PeriodEnd != "" {
		periodEnd, err := time.Parse(time.RFC3339, payload.PeriodEnd)
		if err != nil {
			return nil, fmt.Errorf("parse period_end: %w", err)
		}
		report.PeriodEnd = periodEnd.UTC().Format(time.RFC3339)
	}

	addMetric(report.Metrics, "premium_requests_used", payload.TotalPremiumRequests)
	addMetric(report.Metrics, "premium_requests_limit", payload.PremiumRequestsLimit)
	addMetric(report.Metrics, "premium_requests_remaining", payload.RemainingPremiumRequests)
	addMetric(report.Metrics, "chat_turns", payload.TotalChatTurns)
	addMetric(report.Metrics, "completions", payload.TotalCompletions)
	addMetric(report.Metrics, "acceptances", payload.TotalAcceptanceCount)

	if _, hasRemaining := report.Metrics["premium_requests_remaining"]; !hasRemaining {
		used, hasUsed := report.Metrics["premium_requests_used"]
		limit, hasLimit := report.Metrics["premium_requests_limit"]
		if hasUsed && hasLimit {
			report.Metrics["premium_requests_remaining"] = limit - used
		}
	}

	return report, nil
}

func EnvTokenResolver(_ context.Context) (string, error) {
	for _, envVar := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		token := strings.TrimSpace(os.Getenv(envVar))
		if token != "" {
			return token, nil
		}
	}

	return "", errors.New("missing GitHub token: set GITHUB_TOKEN or GH_TOKEN with the copilot scope")
}

func addMetric(metrics map[string]float64, key string, value *float64) {
	if value != nil {
		metrics[key] = *value
	}
}
