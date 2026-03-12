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

package v1alpha3

import (
	"fmt"
	"slices"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

var validDays = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

// validatePeriod validates a single ScalerPeriod configuration.
func validatePeriod(p common.ScalerPeriod, index int) error {
	if p.Type != "up" && p.Type != "down" {
		return fmt.Errorf("periods[%d].type must be 'up' or 'down', got %q", index, p.Type)
	}

	if p.Time.Recurring == nil && p.Time.Fixed == nil {
		return fmt.Errorf("periods[%d].time must have either 'recurring' or 'fixed'", index)
	}

	if p.Time.Recurring != nil && p.Time.Fixed != nil {
		return fmt.Errorf("periods[%d].time must have either 'recurring' or 'fixed', not both", index)
	}

	if p.Time.Recurring != nil {
		if err := validateRecurringPeriod(p.Time.Recurring, index); err != nil {
			return err
		}
	}

	if p.MinReplicas != nil && p.MaxReplicas != nil && *p.MinReplicas > *p.MaxReplicas {
		return fmt.Errorf("periods[%d].minReplicas (%d) must be <= maxReplicas (%d)", index, *p.MinReplicas, *p.MaxReplicas)
	}

	return nil
}

func validateRecurringPeriod(r *common.RecurringPeriod, periodIndex int) error {
	if len(r.Days) == 0 {
		return fmt.Errorf("periods[%d].time.recurring.days must not be empty", periodIndex)
	}

	for _, day := range r.Days {
		if !slices.Contains(validDays, day) {
			return fmt.Errorf("periods[%d].time.recurring.days contains invalid day %q", periodIndex, day)
		}
	}

	if r.StartTime == "" {
		return fmt.Errorf("periods[%d].time.recurring.startTime is required", periodIndex)
	}

	if r.EndTime == "" {
		return fmt.Errorf("periods[%d].time.recurring.endTime is required", periodIndex)
	}

	return nil
}
