// Package utils provides constants for GCP resource management in the kubecloudscaler project.
package utils

import kubeconsts "github.com/kubecloudscaler/kubecloudscaler/pkg/consts"

// Re-export shared constants for backward compatibility.
const (
	AnnotationsPrefix       = kubeconsts.AnnotationsPrefix
	AnnotationsOrigValue    = kubeconsts.AnnotationsOrigValue
	AnnotationsMinOrigValue = kubeconsts.AnnotationsMinOrigValue
	AnnotationsMaxOrigValue = kubeconsts.AnnotationsMaxOrigValue
	PeriodType              = kubeconsts.PeriodType
	PeriodStartTime         = kubeconsts.PeriodStartTime
	PeriodEndTime           = kubeconsts.PeriodEndTime
	PeriodTimezone          = kubeconsts.PeriodTimezone
	FieldManager            = kubeconsts.FieldManager
)

// GCP-specific constants.
const (
	// DefaultResource is the default GCP resource type for scaling.
	DefaultResource = "vm-instances"
	// InstanceRunning is the GCP instance running state.
	InstanceRunning = "RUNNING"
	// InstanceStopped is the GCP instance stopped state.
	InstanceStopped = "TERMINATED"
	// InstanceStopping is the GCP instance stopping state.
	InstanceStopping = "STOPPING"
	// InstanceStarting is the GCP instance starting state.
	InstanceStarting = "STARTING"
)
