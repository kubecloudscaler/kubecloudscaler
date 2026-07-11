// Package cnpg provides utility functions for CloudNativePG Cluster resource management.
package cnpg

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// New creates a new CloudNativePG Cluster resource manager.
func New(ctx context.Context, config *utils.Config) (*Cnpg, error) {
	logger := zerolog.Ctx(ctx)
	clientAdapter := utils.NewKubernetesClientAdapter(config.Client)
	namespaceMgr := utils.NewNamespaceManager(clientAdapter, *logger, nil)

	k8sResource, err := namespaceMgr.InitConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error initializing k8s config: %w", err)
	}

	resource := &Cnpg{
		Resource: k8sResource,
		Logger:   logger,
	}

	resource.init(config.DynamicClient)

	return resource, nil
}
