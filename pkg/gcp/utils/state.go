// Package utils provides utility functions for GCP resource management in the kubecloudscaler project.
package utils

import (
	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// GetDesiredState returns the target instance status string for the given period.
// stoppedState is the resource-specific constant: gcpUtils.InstanceStopped ("TERMINATED") for
// plain VM instances, or the MIG-specific "STOPPED" for instance-group-managers — the two
// resource types use distinct proto enums for their stopped state.
func GetDesiredState(p *periodPkg.Period, defaultPeriodType, stoppedState string) string {
	defaultState := stoppedState
	if defaultPeriodType == string(common.PeriodTypeUp) {
		defaultState = InstanceRunning
	}

	if p == nil {
		return defaultState
	}

	switch p.Type {
	case common.PeriodTypeUp:
		return InstanceRunning
	case common.PeriodTypeDown:
		return stoppedState
	default:
		return defaultState
	}
}
