//nolint:nolintlint,revive // package name 'utils' is acceptable for K8s utility functions
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
	// AnnotationIgnore is the annotation key for ignoring the resource.
	AnnotationIgnore = "ignore"
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
)
