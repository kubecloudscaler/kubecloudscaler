package computeinstances

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

// New creates a new ComputeInstances resource handler
func New(ctx context.Context, config *gcpUtils.Config) (*ComputeInstances, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Client == nil {
		return nil, fmt.Errorf("GCP client cannot be nil")
	}

	if config.ProjectId == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	return &ComputeInstances{
		Config: config,
		Period: config.Period,
		Logger: zerolog.Ctx(ctx),
	}, nil
}
