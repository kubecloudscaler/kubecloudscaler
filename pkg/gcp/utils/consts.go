// Package utils provides constants for GCP resource management in the kubecloudscaler project.
package utils

const (
	// AnnotationsPrefix is the prefix for kubecloudscaler annotations.
	AnnotationsPrefix = "kubecloudscaler.cloud"
	// AnnotationsOrigValue is the annotation key for original values.
	AnnotationsOrigValue = "original-value"
	// AnnotationsMinOrigValue is the annotation key for minimum original values.
	AnnotationsMinOrigValue = "min-original-value"
	// AnnotationsMaxOrigValue is the annotation key for maximum original values.
	AnnotationsMaxOrigValue = "max-original-value"
	// PeriodType is the annotation key for period type.
	PeriodType = "period-type"
	// PeriodStartTime is the annotation key for period start time.
	PeriodStartTime = "period-start-time"
	// PeriodEndTime is the annotation key for period end time.
	PeriodEndTime = "period-end-time"
	// PeriodTimezone is the annotation key for period timezone.
	PeriodTimezone = "period-timezone"
	// FieldManager is the field manager name for Kubernetes resources.
	FieldManager = "kubecloudscaler"

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
