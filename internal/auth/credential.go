// Package auth provides credential types and an in-process credential store
// shared across provider adapters.
package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Source identifies where a credential came from.
type Source string

const (
	SourceDeviceFlow Source = "github-device-flow"
	SourceEnvVar     Source = "env-var"
)

// Credential holds a token and its provenance.
type Credential struct {
	Token     string    `json:"token"`
	Source    Source    `json:"source"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"` // zero means no expiry
}

// Valid reports whether the credential has a non-empty token and has not expired.
func (c Credential) Valid() bool {
	if c.Token == "" {
		return false
	}
	if !c.ExpiresAt.IsZero() && time.Now().After(c.ExpiresAt) {
		return false
	}
	return true
}

// Store is a thread-safe credential registry keyed by provider ID.
// It can optionally persist to a JSON file via Load/Save.
type Store struct {
	mu    sync.RWMutex
	creds map[string]Credential
}

// NewStore returns an empty credential store.
func NewStore() *Store {
	return &Store{creds: make(map[string]Credential)}
}

// Set stores a credential for a provider key, replacing any existing one.
func (s *Store) Set(providerKey string, cred Credential) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creds[providerKey] = cred
}

// Get returns the credential for a provider key and whether it was found.
func (s *Store) Get(providerKey string) (Credential, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.creds[providerKey]
	return c, ok
}

// Clear removes the credential for a provider key.
func (s *Store) Clear(providerKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.creds, providerKey)
}

// Load reads credentials from a JSON file into the store.
// If the file does not exist the call is a no-op (not an error).
func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read credentials file: %w", err)
	}

	var persisted map[string]Credential
	if err := json.Unmarshal(data, &persisted); err != nil {
		// Corrupt file — log and treat as empty rather than hard-failing.
		log.Printf("gastank: credentials file %s is corrupt and will be ignored: %v", path, err)
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range persisted {
		if v.Valid() {
			s.creds[k] = v
		}
	}
	return nil
}

// Save writes the current credentials to a JSON file, creating parent
// directories as needed.
func (s *Store) Save(path string) error {
	s.mu.RLock()
	// Copy to avoid holding the lock during I/O.
	snapshot := make(map[string]Credential, len(s.creds))
	for k, v := range s.creds {
		snapshot[k] = v
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	// Write to a temp file then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename credentials file: %w", err)
	}
	return nil
}

// DefaultCredentialsPath returns the platform-appropriate path for the
// credentials file: <os.UserConfigDir>/gastank/credentials.json.
func DefaultCredentialsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(dir, "gastank", "credentials.json"), nil
}
