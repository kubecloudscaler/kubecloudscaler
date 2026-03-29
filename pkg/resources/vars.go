// Package resources provides variables and constants for resource management.
package resources

import (
	"errors"
	"slices"
)

var (
	// ErrResourceNotFound is returned when a requested resource is not found.
	ErrResourceNotFound = errors.New("resource not found")
)

// availableResources is the backing list of resource types. Use GetAvailableResources() for immutable access.
var availableResources = []string{
	"deployments",
	"statefulsets",
	"cronjobs",
	"github-ars",
	"scaledobjects",
}

// GetAvailableResources returns a copy of the available resource types for scaling.
// Callers cannot mutate the returned slice.
func GetAvailableResources() []string {
	return slices.Clone(availableResources)
}
