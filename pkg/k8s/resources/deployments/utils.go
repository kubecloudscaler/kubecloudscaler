package deployments

import (
	"context"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func New(ctx context.Context, config *utils.Config) (*Deployments, error) {
	k8sResource, err := utils.InitConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	resource := &Deployments{
		Resource: k8sResource,
	}

	resource.init(config.Client)

	return resource, nil
}
