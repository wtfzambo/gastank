package usage

import (
	"context"
	"fmt"
	"sort"
)

// Service is a tiny provider registry for fetching usage by name.
type Service struct {
	providers map[string]Provider
}

func NewService(providers ...Provider) *Service {
	registry := make(map[string]Provider, len(providers))
	for _, provider := range providers {
		registry[provider.Name()] = provider
	}

	return &Service{providers: registry}
}

func (s *Service) Fetch(ctx context.Context, providerName string) (*UsageReport, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}

	return provider.FetchUsage(ctx)
}

func (s *Service) Providers() []string {
	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}
