// Package auth provides credential types and an in-process credential store
// shared across provider adapters.
package auth

import (
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
	Token     string
	Source    Source
	ExpiresAt time.Time // zero means no expiry
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

// Store is a thread-safe in-memory registry of credentials keyed by
// a provider ID (e.g. "github-copilot").
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
