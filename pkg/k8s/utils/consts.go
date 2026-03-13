//nolint:nolintlint,revive // package name 'utils' is acceptable for K8s utility functions
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

	// AnnotationIgnore is the annotation key for ignoring the resource (K8s-specific).
	AnnotationIgnore = "ignore"
)
