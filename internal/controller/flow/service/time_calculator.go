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
	"github.com/rs/zerolog"
)

// TimeCalculatorService handles time-related calculations for periods
type TimeCalculatorService struct {
	logger *zerolog.Logger
}

// NewTimeCalculatorService creates a new TimeCalculatorService
func NewTimeCalculatorService(logger *zerolog.Logger) *TimeCalculatorService {
	return &TimeCalculatorService{
		logger: logger,
	}
}

// CalculatePeriodStartTime calculates the start time for a period with delay
func (t *TimeCalculatorService) CalculatePeriodStartTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error) {
	baseStartTime, err := t.parsePeriodStartTime(period)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse period start time: %w", err)
	}

	return baseStartTime.Add(delay), nil
}

// CalculatePeriodEndTime calculates the end time for a period with delay
func (t *TimeCalculatorService) CalculatePeriodEndTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error) {
	baseEndTime, err := t.parsePeriodEndTime(period)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse period end time: %w", err)
	}

	return baseEndTime.Add(-delay), nil
}

// GetPeriodDuration calculates the duration of a period
func (t *TimeCalculatorService) GetPeriodDuration(period *common.ScalerPeriod) (time.Duration, error) {
	startTime, err := t.parsePeriodStartTime(period)
	if err != nil {
		return 0, fmt.Errorf("failed to parse start time: %w", err)
	}

	endTime, err := t.parsePeriodEndTime(period)
	if err != nil {
		return 0, fmt.Errorf("failed to parse end time: %w", err)
	}

	if endTime.Before(startTime) {
		return 0, fmt.Errorf("end time is before start time")
	}

	return endTime.Sub(startTime), nil
}

// parsePeriodStartTime parses the start time from a period
func (t *TimeCalculatorService) parsePeriodStartTime(period *common.ScalerPeriod) (time.Time, error) {
	if period.Time.Recurring != nil {
		return time.Parse("15:04", period.Time.Recurring.StartTime)
	}

	if period.Time.Fixed != nil {
		return time.Parse("2006-01-02 15:04:05", period.Time.Fixed.StartTime)
	}

	return time.Time{}, fmt.Errorf("no valid time period found")
}

// parsePeriodEndTime parses the end time from a period
func (t *TimeCalculatorService) parsePeriodEndTime(period *common.ScalerPeriod) (time.Time, error) {
	if period.Time.Recurring != nil {
		return time.Parse("15:04", period.Time.Recurring.EndTime)
	}

	if period.Time.Fixed != nil {
		return time.Parse("2006-01-02 15:04:05", period.Time.Fixed.EndTime)
	}

	return time.Time{}, fmt.Errorf("no valid end time found")
}
