package utils_test

import (
	"testing"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

func TestGetDesiredState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		p                 *period.Period
		defaultPeriodType string
		stoppedState      string
		want              string
	}{
		{
			name:              "nil period uses stopped default",
			p:                 nil,
			defaultPeriodType: "down",
			stoppedState:      utils.InstanceStopped,
			want:              utils.InstanceStopped,
		},
		{
			name:              "nil period with defaultPeriodType=up uses running",
			p:                 nil,
			defaultPeriodType: string(common.PeriodTypeUp),
			stoppedState:      utils.InstanceStopped,
			want:              utils.InstanceRunning,
		},
		{
			name:              "period type up returns running",
			p:                 &period.Period{Type: common.PeriodTypeUp},
			defaultPeriodType: "down",
			stoppedState:      utils.InstanceStopped,
			want:              utils.InstanceRunning,
		},
		{
			name:              "period type down returns stoppedState",
			p:                 &period.Period{Type: common.PeriodTypeDown},
			defaultPeriodType: "down",
			stoppedState:      utils.InstanceStopped,
			want:              utils.InstanceStopped,
		},
		{
			name:              "period type down respects custom stoppedState",
			p:                 &period.Period{Type: common.PeriodTypeDown},
			defaultPeriodType: "down",
			stoppedState:      "STOPPED",
			want:              "STOPPED",
		},
		{
			name:              "unknown period type falls back to default",
			p:                 &period.Period{Type: common.PeriodType("noaction")},
			defaultPeriodType: "down",
			stoppedState:      utils.InstanceStopped,
			want:              utils.InstanceStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := utils.GetDesiredState(tt.p, tt.defaultPeriodType, tt.stoppedState)
			if got != tt.want {
				t.Errorf("GetDesiredState() = %q, want %q", got, tt.want)
			}
		})
	}
}
