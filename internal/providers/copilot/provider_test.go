package copilot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gastank/internal/auth"
)

// sampleResponse mirrors the /copilot_internal/user shape for tests.
const sampleResponse = `{
	"copilot_plan": "pro",
	"quota_reset_date": "2026-04-01",
	"quota_snapshots": {
		"premium_interactions": {
			"percent_remaining": 85.0,
			"remaining": 255,
			"quota_remaining": 255,
			"unlimited": false,
			"timestamp_utc": "2026-03-28T18:00:00Z"
		},
		"chat": {
			"percent_remaining": 90.0,
			"remaining": 900,
			"quota_remaining": 900,
			"unlimited": false,
			"timestamp_utc": "2026-03-28T18:00:00Z"
		},
		"completions": {
			"unlimited": true
		}
	}
}`

func TestFetchUsageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Must hit the internal endpoint.
		if r.URL.Path != "/copilot_internal/user" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		// Auth header must use "token" scheme.
		if got := r.Header.Get("Authorization"); got != "token test-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if r.Header.Get("Editor-Version") == "" {
			t.Fatal("missing Editor-Version header")
		}
		if r.Header.Get("X-Github-Api-Version") == "" {
			t.Fatal("missing X-Github-Api-Version header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer server.Close()

	provider := NewProvider(Config{
		BaseURL: server.URL,
		TokenResolver: func(context.Context) (string, error) {
			return "test-token", nil
		},
	})

	report, err := provider.FetchUsage(context.Background())
	if err != nil {
		t.Fatalf("FetchUsage() unexpected error: %v", err)
	}
	if report.Provider != ProviderName {
		t.Fatalf("provider name: want %q, got %q", ProviderName, report.Provider)
	}
	if got := report.Metadata["plan"]; got != "pro" {
		t.Fatalf("plan metadata: want %q, got %q", "pro", got)
	}
	if got := report.Metadata["quota_reset_date"]; got != "2026-04-01" {
		t.Fatalf("quota_reset_date: want %q, got %q", "2026-04-01", got)
	}
	if got := report.Metadata["endpoint"]; got != "/copilot_internal/user" {
		t.Fatalf("endpoint: want %q, got %q", "/copilot_internal/user", got)
	}
	if got := report.Metrics["premium_percent_remaining"]; got != 85.0 {
		t.Fatalf("premium_percent_remaining: want 85.0, got %v", got)
	}
	if got := report.Metrics["premium_remaining"]; got != 255 {
		t.Fatalf("premium_remaining: want 255, got %v", got)
	}
	if got := report.Metrics["chat_percent_remaining"]; got != 90.0 {
		t.Fatalf("chat_percent_remaining: want 90.0, got %v", got)
	}
	if got := report.Metrics["completions_unlimited"]; got != 1 {
		t.Fatalf("completions_unlimited: want 1, got %v", got)
	}
	if _, found := report.Metrics["completions_percent_remaining"]; found {
		t.Fatal("completions_percent_remaining should not be set when quota is unlimited")
	}
}

func TestFetchUsageHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := NewProvider(Config{
		BaseURL: server.URL,
		TokenResolver: func(context.Context) (string, error) {
			return "test-token", nil
		},
	})

	_, err := provider.FetchUsage(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !containsAll(err.Error(), []string{"401"}) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchUsageInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"copilot_plan":`))
	}))
	defer server.Close()

	provider := NewProvider(Config{
		BaseURL: server.URL,
		TokenResolver: func(context.Context) (string, error) {
			return "test-token", nil
		},
	})

	_, err := provider.FetchUsage(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// --- StoreTokenResolver tests ---

func TestStoreTokenResolverUsesStore(t *testing.T) {
	store := auth.NewStore()
	store.Set(ProviderName, auth.Credential{Token: "store-token", Source: auth.SourceDeviceFlow})

	resolver := StoreTokenResolver(store)
	got, err := resolver(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "store-token" {
		t.Fatalf("expected store-token, got %q", got)
	}
}

func TestStoreTokenResolverNoCredentials(t *testing.T) {
	store := auth.NewStore() // empty — no credentials

	resolver := StoreTokenResolver(store)
	_, err := resolver(context.Background())
	if err == nil {
		t.Fatal("expected error when store is empty")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error should mention 'not authenticated', got: %v", err)
	}
}

func TestNewProviderNoStoreReturnsError(t *testing.T) {
	// No CredStore, no TokenResolver — should return a clear error.
	p := NewProvider(Config{})
	_, err := p.tokenResolver(context.Background())
	if err == nil {
		t.Fatal("expected error when no store and no resolver configured")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error should mention 'not authenticated', got: %v", err)
	}
}

func TestNewProviderUsesCredStore(t *testing.T) {
	store := auth.NewStore()
	store.Set(ProviderName, auth.Credential{Token: "stored-tok", Source: auth.SourceDeviceFlow})

	// No TokenResolver set — should auto-build from CredStore.
	p := NewProvider(Config{CredStore: store})
	tok, err := p.tokenResolver(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "stored-tok" {
		t.Fatalf("expected stored-tok, got %q", tok)
	}
}

func containsAll(haystack string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}
