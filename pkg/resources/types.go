package resources

import (
	"context"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	k8sUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

type Config struct {
	K8s *k8sUtils.Config `json:"k8s,omitempty"`
	GCP *gcpUtils.Config `json:"gcp,omitempty"`
}

type IResource interface {
	SetState(ctx context.Context) ([]kubecloudscalerv1alpha1.ScalerStatusSuccess, []kubecloudscalerv1alpha1.ScalerStatusFailed, error)
}
