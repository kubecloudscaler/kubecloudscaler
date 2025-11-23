// Package cronjobs provides utility functions for CronJob resource management.
package cronjobs

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// New creates a new Cronjobs resource manager.
func New(ctx context.Context, config *utils.Config) (*Cronjobs, error) {
	logger := zerolog.Ctx(ctx)
	clientAdapter := utils.NewKubernetesClientAdapter(config.Client)
	namespaceMgr := utils.NewNamespaceManager(clientAdapter, *logger)

	k8sResource, err := namespaceMgr.InitConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error initializing k8s config: %w", err)
	}

	resource := &Cronjobs{
		Resource: k8sResource,
		Logger:   logger,
	}

	resource.init(config.Client)

	return resource, nil
}
