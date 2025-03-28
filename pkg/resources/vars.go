package resources

import "errors"

var (
	AvailableResources = []string{
		"deployments",
		"statefulsets",
		"cronjobs",
		// "horizontalpodautoscalers",
		// "hpa",
	}
	ErrResourceNotFound = errors.New("resource not found")
)
