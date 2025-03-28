package resources

import (
	"context"
	"fmt"

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
		resource, err = deployments.New(ctx, config.K8s)
		if err != nil {
			return nil, fmt.Errorf("error creating deployments resource: %w", err)
		}

	case "statefulsets":
		resource, err = statefulsets.New(ctx, config.K8s)
		if err != nil {
			return nil, fmt.Errorf("error creating statefulsets resource: %w", err)
		}

	case "cronjobs":
		resource, err = cronjobs.New(ctx, config.K8s)
		if err != nil {
			return nil, fmt.Errorf("error creating cronjobs resource: %w", err)
		}
	// case "horizontalpodautoscalers":
	// case "hpa":
	// 	resource = &horizontalpodautoscalers.HorizontalPodAutoscalers{
	// 		Period: period.K8s,
	// 		Config: config.K8s,
	// 	}
	default:
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, resourceName)
	}

	return resource, nil
}
