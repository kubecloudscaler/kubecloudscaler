/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package consts provides shared constants used across K8s and GCP resource management.
package consts

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
)
