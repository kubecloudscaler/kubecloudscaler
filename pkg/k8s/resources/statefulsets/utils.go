package statefulsets

import (
	"context"

	"github.com/k8scloudscaler/k8scloudscaler/pkg/k8s/utils"
)

func New(ctx context.Context, config *utils.Config) (*Statefulsets, error) {
	k8sResource, err := utils.InitConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	resource := &Statefulsets{
		Resource: k8sResource,
	}

	resource.init(config.Client)

	return resource, nil
}
