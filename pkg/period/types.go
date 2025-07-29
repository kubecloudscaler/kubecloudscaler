package period

import (
	"time"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
)

type Period struct {
	Period           *kubecloudscalerv1alpha1.RecurringPeriod
	Type             string
	IsActive         bool
	Hash             string
	GetStartTime     time.Time
	GetEndTime       time.Time
	StartGracePeriod time.Duration
	EndGracePeriod   time.Duration
	Once             *bool
	MinReplicas      int32
	MaxReplicas      int32
}
