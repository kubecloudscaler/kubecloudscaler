// Package resources provides variables and constants for resource management.
package resources

import "errors"

var (
	// AvailableResources contains the list of available resource types for scaling.
	AvailableResources = []string{
		"deployments",
		"statefulsets",
		"cronjobs",
		"github-ars",
		// "horizontalpodautoscalers",
		// "hpa",
	}
	// ErrResourceNotFound is returned when a requested resource is not found.
	ErrResourceNotFound = errors.New("resource not found")
)
