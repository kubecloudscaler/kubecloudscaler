package utils

const (
	AnnotationsPrefix       = "kubecloudscaler.cloud"
	AnnotationsOrigValue    = "original-value"
	AnnotationsMinOrigValue = "min-original-value"
	AnnotationsMaxOrigValue = "max-original-value"
	PeriodType              = "period-type"
	PeriodStartTime         = "period-start-time"
	PeriodEndTime           = "period-end-time"
	PeriodTimezone          = "period-timezone"
	FieldManager            = "kubecloudscaler"

	// GCP specific constants
	DefaultResource  = "compute-instances"
	InstanceRunning  = "RUNNING"
	InstanceStopped  = "TERMINATED"
	InstanceStopping = "STOPPING"
	InstanceStarting = "STARTING"
)
