package hpa

import (
	"github.com/rs/zerolog"
	v2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

type HorizontalPodAutoscalers struct {
	Resource *utils.K8sResource
	Client   v2.AutoscalingV2Interface
	Logger   *zerolog.Logger
}
