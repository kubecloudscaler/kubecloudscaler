package resources

import (
	"context"
	"fmt"

	"github.com/cloudscalerio/cloudscaler/pkg/k8s/resources/deployments"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
	// "github.com/cloudscalerio/cloudscaler/pkg/k8s/resources/horizontalpodautoscalers"
)

func NewResource(ctx context.Context, resourceName string, config Config) (IResource, error) {
	var (
		resource IResource
		err      error
	)

	switch resourceName {
	case "deployments":
		k8sResource := &utils.K8sResource{
			Config: config.K8s,
		}

		k8sResource.NsList, k8sResource.ListOptions, err = utils.PrepareSearch(ctx, config.K8s)
		if err != nil {
			return nil, err
		}

		resource = &deployments.Deployments{
			Resource: k8sResource,
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
