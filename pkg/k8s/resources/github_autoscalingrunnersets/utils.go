// Package ars provides utility functions for Autoscaling Runner Set resource management.
package ars

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// New creates a new Github Autoscaling Runnersets resource manager.
func New(ctx context.Context, config *utils.Config) (*GithubAutoscalingRunnersets, error) {
	logger := zerolog.Ctx(ctx)
	clientAdapter := utils.NewKubernetesClientAdapter(config.Client)
	namespaceMgr := utils.NewNamespaceManager(clientAdapter, *logger)

	k8sResource, err := namespaceMgr.InitConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error initializing k8s config: %w", err)
	}

	resource := &GithubAutoscalingRunnersets{
		Resource: k8sResource,
		Logger:   logger,
	}

	resource.init(config.DynamicClient)

	return resource, nil
}
