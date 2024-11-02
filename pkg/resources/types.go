package resources

import (
	"context"

	"github.com/golgoth31/cloudscaler/api/common"
	k8sUtils "github.com/golgoth31/cloudscaler/pkg/k8s/utils"
)

type Config struct {
	K8s *k8sUtils.Config `json:"k8s"`
}

type IResource interface {
	SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error)
}
