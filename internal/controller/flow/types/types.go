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

// Package types holds reconciliation types shared within the Flow controller.
package types

import (
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// PeriodWithDelay contains period information with calculated delay
type PeriodWithDelay struct {
	Period         common.ScalerPeriod
	StartTimeDelay time.Duration
	EndTimeDelay   time.Duration
	StartTime      time.Time
	EndTime        time.Time
}

// ResourceInfo contains information about a resource and its associated periods.
// Exactly one of K8sRes or GcpRes is set, determined by Type.
type ResourceInfo struct {
	Type    string                               // "k8s" or "gcp"
	K8sRes  *kubecloudscalerv1alpha3.K8sResource // Set when Type == "k8s"
	GcpRes  *kubecloudscalerv1alpha3.GcpResource // Set when Type == "gcp"
	Periods []PeriodWithDelay                    // Associated periods with delays
}
