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
	"fmt"
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/rs/zerolog"
)

// FlowValidatorService handles validation of flow configurations
type FlowValidatorService struct {
	timeCalculator TimeCalculator
	logger         *zerolog.Logger
}

// NewFlowValidatorService creates a new FlowValidatorService
func NewFlowValidatorService(timeCalculator TimeCalculator, logger *zerolog.Logger) *FlowValidatorService {
	return &FlowValidatorService{
		timeCalculator: timeCalculator,
		logger:         logger,
	}
}

// ExtractFlowData extracts all resource names and period names from flows
func (v *FlowValidatorService) ExtractFlowData(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error) {
	resourceNames := make(map[string]bool)
	periodNames := make(map[string]bool)

	for _, flowItem := range flow.Spec.Flows {
		periodNames[flowItem.PeriodName] = true

		for _, resource := range flowItem.Resources {
			resourceNames[resource.Name] = true
		}
	}

	return resourceNames, periodNames, nil
}

// ValidatePeriodTimings validates that the sum of delays for each period doesn't exceed the period duration
func (v *FlowValidatorService) ValidatePeriodTimings(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error {
	periodsMap := v.createPeriodsMap(flow)

	for periodName := range periodNames {
		period, exists := periodsMap[periodName]
		if !exists {
			return fmt.Errorf("period %s referenced in flows but not defined", periodName)
		}

		totalDelay, err := v.calculateTotalDelay(flow, periodName)
		if err != nil {
			return fmt.Errorf("failed to calculate total delay for period %s: %w", periodName, err)
		}

		periodDuration, err := v.timeCalculator.GetPeriodDuration(&period)
		if err != nil {
			return fmt.Errorf("failed to get period duration for %s: %w", periodName, err)
		}

		if totalDelay > periodDuration {
			return fmt.Errorf("total delay %v for period %s exceeds period duration %v",
				totalDelay, periodName, periodDuration)
		}
	}

	return nil
}

// createPeriodsMap creates a map of period names to periods
func (v *FlowValidatorService) createPeriodsMap(flow *kubecloudscalerv1alpha3.Flow) map[string]common.ScalerPeriod {
	periodsMap := make(map[string]common.ScalerPeriod)
	for i := range flow.Spec.Periods {
		period := flow.Spec.Periods[i]
		periodsMap[period.Name] = period
	}
	return periodsMap
}

// calculateTotalDelay calculates the total delay for a specific period
func (v *FlowValidatorService) calculateTotalDelay(flow *kubecloudscalerv1alpha3.Flow, periodName string) (time.Duration, error) {
	var totalDelay time.Duration

	for _, flowItem := range flow.Spec.Flows {
		if flowItem.PeriodName != periodName {
			continue
		}

		for _, resource := range flowItem.Resources {
			if resource.StartTimeDelay != "" {
				delay, err := time.ParseDuration(resource.StartTimeDelay)
				if err != nil {
					return 0, fmt.Errorf("invalid start time delay format for resource %s: %w", resource.Name, err)
				}
				totalDelay += delay
			}

			if resource.EndTimeDelay != "" {
				delay, err := time.ParseDuration(resource.EndTimeDelay)
				if err != nil {
					return 0, fmt.Errorf("invalid end time delay format for resource %s: %w", resource.Name, err)
				}
				totalDelay += delay
			}
		}
	}

	return totalDelay, nil
}
