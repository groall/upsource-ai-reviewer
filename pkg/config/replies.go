package config

import "fmt"

// Replies controls follow-up replies posted to existing Upsource discussions.
type Replies struct {
	ActiveProvider string `yaml:"activeProvider"`
	Enabled        bool   `yaml:"enabled"`
	MaxPerThread   int    `yaml:"maxPerThread"`
	SystemMessage  string `yaml:"systemMessage"`
}

// Validate validates the reply configuration when replies are enabled.
func (r *Replies) Validate() error {
	if !r.Enabled {
		return nil
	}

	if r.MaxPerThread <= 0 {
		return fmt.Errorf("replies.maxPerThread must be > 0 when replies.enabled is true")
	}
	if r.SystemMessage == "" {
		return fmt.Errorf("replies.systemMessage is required when replies.enabled is true")
	}
	// Validate Replies Provider if set (not strictly required to have a providerId, but must be valid if provided)
	if r.ActiveProvider != "" {
		if err := CheckProviderByID(r.ActiveProvider); err != nil {
			return fmt.Errorf("replies %w", err)
		}
		if r.ActiveProvider == ProviderAgent {
			return fmt.Errorf("replies.activeProvider cannot be agent")
		}
	} else {
		return fmt.Errorf("replies.activeProvider is required")
	}

	return nil
}
