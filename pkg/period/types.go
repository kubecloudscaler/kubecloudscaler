package period

import (
	"time"

	k8scloudscalerv1alpha1 "github.com/k8scloudscaler/k8scloudscaler/api/v1alpha1"
)

type Period struct {
	Period       *k8scloudscalerv1alpha1.RecurringPeriod
	Type         string
	IsActive     bool
	Hash         string
	GetStartTime time.Time
	GetEndTime   time.Time
	GracePeriod  time.Duration
	Once         *bool
	MinReplicas  int32
	MaxReplicas  int32
}
