package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ingo/internal/auth"
)

func TestStartDeviceFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/device/code" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected Accept: application/json, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"device_code":      "dev123",
			"user_code":        "ABCD-EFGH",
			"verification_uri": "https://github.com/login/device",
			"expires_in":       900,
			"interval":         5
		}`))
	}))
	defer server.Close()

	flow := newTestFlow(server, t)
	resp, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if resp.DeviceCode != "dev123" {
		t.Fatalf("device_code: want dev123, got %q", resp.DeviceCode)
	}
	if resp.UserCode != "ABCD-EFGH" {
		t.Fatalf("user_code: want ABCD-EFGH, got %q", resp.UserCode)
	}
	if resp.VerificationURI != "https://github.com/login/device" {
		t.Fatalf("verification_uri mismatch: %q", resp.VerificationURI)
	}
}

func TestPollDeviceFlow_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"gho_abc123","token_type":"bearer","scope":"read:user"}`))
	}))
	defer server.Close()

	flow := newTestFlow(server, t)
	cred, err := flow.Poll(context.Background(), "dev123")
	if err != nil {
		t.Fatalf("Poll() unexpected error: %v", err)
	}
	if cred.Token != "gho_abc123" {
		t.Fatalf("token: want gho_abc123, got %q", cred.Token)
	}
	if cred.Source != auth.SourceDeviceFlow {
		t.Fatalf("source: want %q, got %q", auth.SourceDeviceFlow, cred.Source)
	}
}

func TestPollDeviceFlow_Pending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"authorization_pending","error_description":"not yet"}`))
	}))
	defer server.Close()

	flow := newTestFlow(server, t)
	_, err := flow.Poll(context.Background(), "dev123")
	if !errors.Is(err, ErrAuthorizationPending) {
		t.Fatalf("expected ErrAuthorizationPending, got %v", err)
	}
}

func TestPollDeviceFlow_SlowDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"slow_down"}`))
	}))
	defer server.Close()

	flow := newTestFlow(server, t)
	_, err := flow.Poll(context.Background(), "dev123")
	if !errors.Is(err, ErrSlowDown) {
		t.Fatalf("expected ErrSlowDown, got %v", err)
	}
}

func TestPollDeviceFlow_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"expired_token"}`))
	}))
	defer server.Close()

	flow := newTestFlow(server, t)
	_, err := flow.Poll(context.Background(), "dev123")
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestPollDeviceFlow_AccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"access_denied"}`))
	}))
	defer server.Close()

	flow := newTestFlow(server, t)
	_, err := flow.Poll(context.Background(), "dev123")
	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
}

// newTestFlow creates a DeviceFlow that points at the given test server.
// It monkey-patches the URL constants by using a custom httpClient that
// rewrites requests to the test server.
func newTestFlow(server *httptest.Server, t *testing.T) *DeviceFlow {
	t.Helper()
	client := server.Client()
	// Wrap transport to rewrite the host to the test server.
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = &rewriteTransport{base: base, target: server.URL}
	return &DeviceFlow{httpClient: client, clientID: "test-client-id"}
}

type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	parsed, _ := parseURL(rt.target)
	cloned.URL.Scheme = parsed.Scheme
	cloned.URL.Host = parsed.Host
	return rt.base.RoundTrip(cloned)
}

func parseURL(raw string) (struct{ Scheme, Host string }, error) {
	var tmp struct {
		Scheme, Host string
	}
	// simple manual parse — avoids importing net/url just for a test helper
	data, err := json.Marshal(map[string]string{"url": raw})
	_ = data
	_ = err
	// actually use net/url through the standard library
	req, _ := http.NewRequest(http.MethodGet, raw, nil)
	tmp.Scheme = req.URL.Scheme
	tmp.Host = req.URL.Host
	return tmp, nil
}
