package copilot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchUsageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/copilot/usage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "copilot_plan":"pro",
            "period_start":"2026-03-01T00:00:00Z",
            "period_end":"2026-04-01T00:00:00Z",
            "total_premium_requests":42,
            "premium_requests_limit":300,
            "total_chat_turns":19,
            "total_completions":128,
            "total_acceptance_count":91
        }`))
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
		t.Fatalf("FetchUsage() error = %v", err)
	}

	if report.Provider != ProviderName {
		t.Fatalf("expected provider %q, got %q", ProviderName, report.Provider)
	}

	if got := report.Metadata["plan"]; got != "pro" {
		t.Fatalf("expected plan metadata %q, got %q", "pro", got)
	}

	if got := report.Metrics["premium_requests_used"]; got != 42 {
		t.Fatalf("expected premium_requests_used 42, got %v", got)
	}

	if got := report.Metrics["premium_requests_limit"]; got != 300 {
		t.Fatalf("expected premium_requests_limit 300, got %v", got)
	}

	if got := report.Metrics["premium_requests_remaining"]; got != 258 {
		t.Fatalf("expected premium_requests_remaining 258, got %v", got)
	}

	if got := report.Metrics["chat_turns"]; got != 19 {
		t.Fatalf("expected chat_turns 19, got %v", got)
	}

	if report.PeriodStart != "2026-03-01T00:00:00Z" {
		t.Fatalf("unexpected period start: %q", report.PeriodStart)
	}

	if report.PeriodEnd != "2026-04-01T00:00:00Z" {
		t.Fatalf("unexpected period end: %q", report.PeriodEnd)
	}
}

func TestFetchUsageHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
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

	if got := err.Error(); got == "" || !containsAll(got, []string{"403", "forbidden"}) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchUsageInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total_premium_requests":`))
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
}

func containsAll(haystack string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}
