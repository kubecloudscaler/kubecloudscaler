package period

import (
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

type Period struct {
	Period       *common.RecurringPeriod
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
