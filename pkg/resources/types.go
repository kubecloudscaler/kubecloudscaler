package resources

import (
	"context"

	cloudscaleriov1alpha1 "github.com/cloudscalerio/cloudscaler/api/v1alpha1"
	k8sUtils "github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
)

type Config struct {
	K8s *k8sUtils.Config `json:"k8s"`
}

type IResource interface {
	SetState(ctx context.Context) ([]cloudscaleriov1alpha1.ScalerStatusSuccess, []cloudscaleriov1alpha1.ScalerStatusFailed, error)
}
