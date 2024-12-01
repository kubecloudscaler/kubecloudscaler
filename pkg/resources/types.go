package resources

import (
	"context"

	k8scloudscalerv1alpha1 "github.com/k8scloudscaler/k8scloudscaler/api/v1alpha1"
	k8sUtils "github.com/k8scloudscaler/k8scloudscaler/pkg/k8s/utils"
)

type Config struct {
	K8s *k8sUtils.Config `json:"k8s"`
}

type IResource interface {
	SetState(ctx context.Context) ([]k8scloudscalerv1alpha1.ScalerStatusSuccess, []k8scloudscalerv1alpha1.ScalerStatusFailed, error)
}
