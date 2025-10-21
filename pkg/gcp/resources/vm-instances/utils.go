// Package vminstances provides utility functions for VM instance management in GCP.
package vminstances

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

// New creates a new VMnstances resource handler
func New(ctx context.Context, config *gcpUtils.Config) (*VMnstances, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Client == nil {
		return nil, fmt.Errorf("GCP client cannot be nil")
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	return &VMnstances{
		Config: config,
		Period: config.Period,
		Logger: zerolog.Ctx(ctx),
	}, nil
}
