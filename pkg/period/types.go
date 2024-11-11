package period

import (
	"time"

	cloudscaleriov1alpha1 "github.com/cloudscalerio/cloudscaler/api/v1alpha1"
)

type Period struct {
	Period       *cloudscaleriov1alpha1.RecurringPeriod
	Type         string
	IsActive     bool
	Hash         string
	GetStartTime time.Time
	GetEndTime   time.Time
	Once         *bool
	MinReplicas  *int32
	MaxReplicas  *int32
}
