package resources

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	vminstances "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/resources/vm-instances"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/cronjobs"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/deployments"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/statefulsets"
	// "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/horizontalpodautoscalers"
)

func NewResource(resourceName string, config Config, logger *zerolog.Logger) (IResource, error) {
	ctx := context.Background()
	ctx = logger.With().Str("resource-type", resourceName).Logger().WithContext(ctx)
	var (
		resource IResource
		err      error
	)

	switch resourceName {
	case "deployments":
		if config.K8s == nil {
			return nil, fmt.Errorf("K8s config is required for deployments resource")
		}
		resource, err = deployments.New(ctx, config.K8s)
		if err != nil {
			return nil, fmt.Errorf("error creating deployments resource: %w", err)
		}

	case "statefulsets":
		if config.K8s == nil {
			return nil, fmt.Errorf("K8s config is required for statefulsets resource")
		}
		resource, err = statefulsets.New(ctx, config.K8s)
		if err != nil {
			return nil, fmt.Errorf("error creating statefulsets resource: %w", err)
		}

	case "cronjobs":
		if config.K8s == nil {
			return nil, fmt.Errorf("K8s config is required for cronjobs resource")
		}
		resource, err = cronjobs.New(ctx, config.K8s)
		if err != nil {
			return nil, fmt.Errorf("error creating cronjobs resource: %w", err)
		}

	case "vm-instances":
		if config.GCP == nil {
			return nil, fmt.Errorf("GCP config is required for vm-instances resource")
		}
		resource, err = vminstances.New(ctx, config.GCP)
		if err != nil {
			return nil, fmt.Errorf("error creating vm-instances resource: %w", err)
		}

	default:
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, resourceName)
	}

	return resource, nil
}
