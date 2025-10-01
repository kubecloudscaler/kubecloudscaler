package cronjobs

import (
	"github.com/rs/zerolog"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

type Cronjobs struct {
	Resource *utils.K8sResource
	Client   v1.BatchV1Interface
	Logger   *zerolog.Logger
}
