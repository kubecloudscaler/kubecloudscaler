// Package utils provides utility functions for internal use in the kubecloudscaler project.
//
//nolint:revive // Package name 'utils' is acceptable for internal utility functions
package utils

import "errors"

var (
	AppsResources        = []string{"deployments", "statefulsets"}
	HpaResources         = []string{"horizontalpodautoscalers", "hpa", "scaledobjects"}
	ErrMixedAppsHPA      = errors.New("mixing apps and hpa resources is not allowed")
	ErrLoadRestorePeriod = errors.New("unable to load restore period")
	ErrLoadPeriod        = errors.New("unable to load period")
	ErrRunOncePeriod     = errors.New("run once period")
)
