package cronjobs

import (
	"context"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func New(ctx context.Context, config *utils.Config) (*Cronjobs, error) {
	k8sResource, err := utils.InitConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	resource := &Cronjobs{
		Resource: k8sResource,
	}

	resource.init(config.Client)

	return resource, nil
}
