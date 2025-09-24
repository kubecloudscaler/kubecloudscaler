package resources

import (
	"context"
	"fmt"

	computeinstances "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/resources/compute-instances"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/cronjobs"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/deployments"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/statefulsets"
	// "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/horizontalpodautoscalers"
)

func NewResource(ctx context.Context, resourceName string, config Config) (IResource, error) {
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

	case "compute-instances":
		if config.GCP == nil {
			return nil, fmt.Errorf("GCP config is required for compute-instances resource")
		}
		resource, err = computeinstances.New(ctx, config.GCP)
		if err != nil {
			return nil, fmt.Errorf("error creating compute-instances resource: %w", err)
		}

	default:
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, resourceName)
	}

	return resource, nil
}
