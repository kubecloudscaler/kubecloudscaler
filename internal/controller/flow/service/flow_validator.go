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

// ExtractFlowData extracts all resource names and period names from flows. Duplicate
// period definitions (same name twice in spec.periods) and in-section duplicate resource
// definitions (same name twice in spec.resources.k8s or spec.resources.gcp) are rejected
// here since silent first-writer-wins would produce unpredictable runtime behaviour.
func (v *FlowValidatorService) ExtractFlowData(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error) {
	if err := v.checkUniquePeriods(flow); err != nil {
		return nil, nil, err
	}
	if err := v.checkUniqueResources(flow); err != nil {
		return nil, nil, err
	}

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

func (v *FlowValidatorService) checkUniquePeriods(flow *kubecloudscalerv1alpha3.Flow) error {
	seen := make(map[string]bool, len(flow.Spec.Periods))
	for i := range flow.Spec.Periods {
		name := flow.Spec.Periods[i].Name
		if seen[name] {
			return NewValidationError(ReasonDuplicatePeriod,
				fmt.Errorf("period %s defined more than once in spec.periods", name))
		}
		seen[name] = true
	}
	return nil
}

func (v *FlowValidatorService) checkUniqueResources(flow *kubecloudscalerv1alpha3.Flow) error {
	seenK8s := make(map[string]bool, len(flow.Spec.Resources.K8s))
	for i := range flow.Spec.Resources.K8s {
		name := flow.Spec.Resources.K8s[i].Name
		if seenK8s[name] {
			return NewValidationError(ReasonDuplicateResource,
				fmt.Errorf("resource %s defined more than once in spec.resources.k8s", name))
		}
		seenK8s[name] = true
	}
	seenGcp := make(map[string]bool, len(flow.Spec.Resources.Gcp))
	for i := range flow.Spec.Resources.Gcp {
		name := flow.Spec.Resources.Gcp[i].Name
		if seenGcp[name] {
			return NewValidationError(ReasonDuplicateResource,
				fmt.Errorf("resource %s defined more than once in spec.resources.gcp", name))
		}
		seenGcp[name] = true
	}
	return nil
}

// ValidatePeriodTimings validates that the sum of delays for each period doesn't exceed
// the period duration. User-config errors are returned as *ValidationError so
// ProcessingHandler can classify them as CriticalError. See the ReasonXxx constants in
// errors.go for the full set of reasons this can emit.
func (v *FlowValidatorService) ValidatePeriodTimings(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error {
	periodsMap := v.createPeriodsMap(flow)

	for periodName := range periodNames {
		period, exists := periodsMap[periodName]
		if !exists {
			return NewValidationError(ReasonUnknownPeriod,
				fmt.Errorf("period %s referenced in flows but not defined", periodName))
		}

		periodDuration, err := v.timeCalculator.GetPeriodDuration(&period)
		if err != nil {
			if IsValidationError(err) {
				return err
			}
			return NewValidationError(ReasonInvalidPeriodDuration,
				fmt.Errorf("failed to get period duration for %s: %w", periodName, err))
		}

		if err := v.validateResourceDelays(flow, periodName, periodDuration); err != nil {
			return err
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

// validateResourceDelays validates that each resource's combined delay doesn't exceed the period duration
func (v *FlowValidatorService) validateResourceDelays(flow *kubecloudscalerv1alpha3.Flow, periodName string, periodDuration time.Duration) error {
	for _, flowItem := range flow.Spec.Flows {
		if flowItem.PeriodName != periodName {
			continue
		}

		for _, resource := range flowItem.Resources {
			var startDelay, endDelay time.Duration

			if resource.StartTimeDelay != "" {
				d, err := time.ParseDuration(resource.StartTimeDelay)
				if err != nil {
					return NewValidationError(ReasonInvalidDelayFormat,
						fmt.Errorf("invalid start time delay format for resource %s: %w", resource.Name, err))
				}
				startDelay = d
			}

			if resource.EndTimeDelay != "" {
				d, err := time.ParseDuration(resource.EndTimeDelay)
				if err != nil {
					return NewValidationError(ReasonInvalidDelayFormat,
						fmt.Errorf("invalid end time delay format for resource %s: %w", resource.Name, err))
				}
				endDelay = d
			}

			// Adjusted window: [start + startDelay, end + endDelay]
			// Adjusted duration = periodDuration - startDelay + endDelay
			// Must be > 0 for the window to remain valid
			adjustedDuration := periodDuration - startDelay + endDelay
			if adjustedDuration <= 0 {
				return NewValidationError(ReasonInvertedWindow, fmt.Errorf(
					"resource %s: adjusted window is invalid (duration %v) for period %s — "+
						"startTimeDelay (%v) and endTimeDelay (%v) invert the period window (duration %v)",
					resource.Name, adjustedDuration, periodName,
					startDelay, endDelay, periodDuration,
				))
			}
		}
	}

	return nil
}
