// Package github implements the GitHub OAuth device flow.
// See: https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow
//
// This flow is the correct choice for desktop/CLI apps because:
//   - No redirect URI is needed.
//   - The user authenticates in their browser and the app polls for completion.
//   - Works offline-first and doesn't depend on the gh CLI.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gastank/internal/auth"
)

// GitHub's public OAuth app for Copilot / VS Code.
// Using the same client ID as GitHub Copilot Chat so the token has the
// right scopes for the /copilot_internal/* endpoints.
const (
	copilotClientID = "Iv1.b507a08c87ecfe98"
	deviceCodeURL   = "https://github.com/login/device/code"
	accessTokenURL  = "https://github.com/login/oauth/access_token"
)

// DeviceCodeResponse is returned by the device-code endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// tokenResponse is the internal poll response shape.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// DeviceFlow manages a single GitHub device-flow authorisation attempt.
type DeviceFlow struct {
	httpClient *http.Client
	clientID   string
}

// NewDeviceFlow creates a DeviceFlow. Pass nil httpClient to use the default.
func NewDeviceFlow(httpClient *http.Client) *DeviceFlow {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &DeviceFlow{httpClient: httpClient, clientID: copilotClientID}
}

// Start requests a device code from GitHub and returns the user-facing
// instructions (user_code + verification_uri) plus the device_code the caller
// must supply when polling.
func (d *DeviceFlow) Start(ctx context.Context) (*DeviceCodeResponse, error) {
	body := url.Values{}
	body.Set("client_id", d.clientID)
	body.Set("scope", "read:user")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build device-code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read device-code response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device-code endpoint returned %s: %s", resp.Status, raw)
	}

	var result DeviceCodeResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode device-code response: %w", err)
	}
	if result.DeviceCode == "" {
		return nil, fmt.Errorf("GitHub returned an empty device_code")
	}
	return &result, nil
}

// ErrAuthorizationPending is returned when the user hasn't approved yet.
var ErrAuthorizationPending = fmt.Errorf("authorization pending")

// ErrSlowDown is returned when the client should back off its poll rate.
var ErrSlowDown = fmt.Errorf("slow down")

// ErrExpired is returned when the device code has expired.
var ErrExpired = fmt.Errorf("device code expired")

// ErrAccessDenied is returned when the user explicitly declined.
var ErrAccessDenied = fmt.Errorf("access denied by user")

// Poll checks whether the user has approved the device flow.
// Returns (credential, nil) on success.
// Returns a sentinel error (ErrAuthorizationPending, ErrSlowDown, ErrExpired,
// ErrAccessDenied) on expected transient/terminal states.
func (d *DeviceFlow) Poll(ctx context.Context, deviceCode string) (auth.Credential, error) {
	body := url.Values{}
	body.Set("client_id", d.clientID)
	body.Set("device_code", deviceCode)
	body.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, accessTokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return auth.Credential{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return auth.Credential{}, fmt.Errorf("poll token endpoint: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return auth.Credential{}, fmt.Errorf("read token response: %w", err)
	}

	var result tokenResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return auth.Credential{}, fmt.Errorf("decode token response: %w", err)
	}

	switch result.Error {
	case "":
		// success
	case "authorization_pending":
		return auth.Credential{}, ErrAuthorizationPending
	case "slow_down":
		return auth.Credential{}, ErrSlowDown
	case "expired_token":
		return auth.Credential{}, ErrExpired
	case "access_denied":
		return auth.Credential{}, ErrAccessDenied
	default:
		return auth.Credential{}, fmt.Errorf("token error %q: %s", result.Error, result.ErrorDesc)
	}

	if result.AccessToken == "" {
		return auth.Credential{}, fmt.Errorf("GitHub returned an empty access_token")
	}

	return auth.Credential{
		Token:  result.AccessToken,
		Source: auth.SourceDeviceFlow,
	}, nil
}
