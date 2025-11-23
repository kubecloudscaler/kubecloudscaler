// Package deployments provides utility functions for Deployment resource management.
package deployments

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// New creates a new Deployments resource manager.
func New(ctx context.Context, config *utils.Config) (*Deployments, error) {
	logger := zerolog.Ctx(ctx)
	clientAdapter := utils.NewKubernetesClientAdapter(config.Client)
	namespaceMgr := utils.NewNamespaceManager(clientAdapter, *logger)

	k8sResource, err := namespaceMgr.InitConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error initializing k8s config: %w", err)
	}

	resource := &Deployments{
		Resource: k8sResource,
		Logger:   logger,
	}

	resource.init(config.Client)

	return resource, nil
}
