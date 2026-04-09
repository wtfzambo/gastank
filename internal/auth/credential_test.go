package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSetGetClear(t *testing.T) {
	store := NewStore()
	cred := Credential{Token: "tok1", Source: SourceDeviceFlow}
	store.Set("github-copilot", cred)

	got, ok := store.Get("github-copilot")
	if !ok {
		t.Fatal("expected credential to be present")
	}
	if got.Token != "tok1" {
		t.Fatalf("token: want tok1, got %q", got.Token)
	}

	store.Clear("github-copilot")
	_, ok = store.Get("github-copilot")
	if ok {
		t.Fatal("expected credential to be gone after Clear")
	}
}

func TestCredentialValid(t *testing.T) {
	// Valid: non-empty token, no expiry.
	c := Credential{Token: "t"}
	if !c.Valid() {
		t.Fatal("expected valid credential")
	}

	// Invalid: empty token.
	c2 := Credential{}
	if c2.Valid() {
		t.Fatal("empty token should be invalid")
	}

	// Invalid: expired.
	c3 := Credential{Token: "t", ExpiresAt: time.Now().Add(-time.Hour)}
	if c3.Valid() {
		t.Fatal("expired credential should be invalid")
	}

	// Valid: not yet expired.
	c4 := Credential{Token: "t", ExpiresAt: time.Now().Add(time.Hour)}
	if !c4.Valid() {
		t.Fatal("not-yet-expired credential should be valid")
	}
}

func TestStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")

	store := NewStore()
	store.Set("github-copilot", Credential{Token: "mytoken", Source: SourceDeviceFlow})
	store.Set("ollama", Credential{Token: "localtoken", Source: SourceEnvVar})

	if err := store.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load into a fresh store.
	store2 := NewStore()
	if err := store2.Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got, ok := store2.Get("github-copilot")
	if !ok {
		t.Fatal("expected github-copilot credential after load")
	}
	if got.Token != "mytoken" {
		t.Fatalf("token: want mytoken, got %q", got.Token)
	}
	if got.Source != SourceDeviceFlow {
		t.Fatalf("source: want %q, got %q", SourceDeviceFlow, got.Source)
	}

	got2, ok2 := store2.Get("ollama")
	if !ok2 {
		t.Fatal("expected ollama credential after load")
	}
	if got2.Token != "localtoken" {
		t.Fatalf("ollama token: want localtoken, got %q", got2.Token)
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	store := NewStore()
	err := store.Load("/nonexistent/path/creds.json")
	if err != nil {
		t.Fatalf("Load of missing file should be no-op, got: %v", err)
	}
}

func TestStoreLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	if err := os.WriteFile(path, []byte("not json {{"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewStore()
	// Should silently ignore corrupt data, not error.
	if err := store.Load(path); err != nil {
		t.Fatalf("Load of corrupt file should be silent no-op, got: %v", err)
	}
}

func TestStoreSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "creds.json")

	store := NewStore()
	store.Set("k", Credential{Token: "v", Source: SourceEnvVar})

	if err := store.Save(path); err != nil {
		t.Fatalf("Save() should create parent dirs, got: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist after Save: %v", err)
	}
}

func TestStoreExpiredCredNotLoaded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")

	store := NewStore()
	// Store an expired credential.
	store.Set("expired-provider", Credential{
		Token:     "old",
		Source:    SourceDeviceFlow,
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	if err := store.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	store2 := NewStore()
	if err := store2.Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	_, ok := store2.Get("expired-provider")
	if ok {
		t.Fatal("expired credential should not be loaded into store")
	}
}
