package resources

import (
	"context"
	"fmt"

	"github.com/cloudscalerio/cloudscaler/pkg/k8s/resources/cronjobs"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/resources/deployments"
	// "github.com/cloudscalerio/cloudscaler/pkg/k8s/resources/horizontalpodautoscalers"
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
			return nil, err
		}

	case "cronjobs":
		resource, err = cronjobs.New(ctx, config.K8s)
		if err != nil {
			return nil, err
		}
	// case "horizontalpodautoscalers":
	// case "hpa":
	// 	resource = &horizontalpodautoscalers.HorizontalPodAutoscalers{
	// 		Period: period.K8s,
	// 		Config: config.K8s,
	// 	}
	default:
		return nil, fmt.Errorf("resource %s not found", resourceName)
	}

	return resource, nil
}
