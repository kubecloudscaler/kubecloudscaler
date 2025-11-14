// Package deployments provides utility functions for Deployment resource management.
package ars

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// New creates a new Github Autoscaling Runnersets resource manager.
func New(ctx context.Context, config *utils.Config) (*GithubAutoscalingRunnersets, error) {
	k8sResource, err := utils.InitConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error initializing k8s config: %w", err)
	}

	resource := &GithubAutoscalingRunnersets{
		Resource: k8sResource,
		Logger:   zerolog.Ctx(ctx),
	}

	resource.init(config.DynamicClient)

	return resource, nil
}
