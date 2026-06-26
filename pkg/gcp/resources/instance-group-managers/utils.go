// Package instancegroupmanagers provides MIG scaling functionality for GCP resources.
package instancegroupmanagers

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

// New creates a new InstanceGroupManagers resource handler.
func New(ctx context.Context, config *gcpUtils.Config) (*InstanceGroupManagers, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Client == nil {
		return nil, fmt.Errorf("GCP client cannot be nil")
	}

	if config.Client.InstanceGroupManagers == nil {
		return nil, fmt.Errorf("GCP InstanceGroupManagers client cannot be nil")
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	if config.LabelSelector != nil {
		return nil, fmt.Errorf(
			"labelSelector is not supported for instance-group-managers: " +
				"MIG resources expose no labels at the resource level; use names instead")
	}

	return &InstanceGroupManagers{
		Config: config,
		Period: config.Period,
		Logger: zerolog.Ctx(ctx),
	}, nil
}
