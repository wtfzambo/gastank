package usage

import "context"

// Provider fetches usage data for a single AI provider.
type Provider interface {
	Name() string
	FetchUsage(ctx context.Context) (*UsageReport, error)
}

// UsageReport is the normalized payload returned by provider adapters.
type UsageReport struct {
	Provider    string             `json:"provider"`
	RetrievedAt string             `json:"retrievedAt"`
	Metrics     map[string]float64 `json:"metrics"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}
