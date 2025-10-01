package statefulsets

import (
	"github.com/rs/zerolog"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

type Statefulsets struct {
	Resource *utils.K8sResource
	Client   v1.AppsV1Interface
	Logger   *zerolog.Logger
}
