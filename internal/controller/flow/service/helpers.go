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

package service

import (
	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// CreatePeriodsMap creates a map of period names to periods from a flow.
// This is a shared helper function used by multiple services.
func CreatePeriodsMap(flow *kubecloudscalerv1alpha3.Flow) map[string]common.ScalerPeriod {
	periodsMap := make(map[string]common.ScalerPeriod)
	for i := range flow.Spec.Periods {
		period := flow.Spec.Periods[i]
		periodsMap[period.Name] = period
	}
	return periodsMap
}
