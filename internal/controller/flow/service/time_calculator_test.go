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
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

func TestTimeCalculatorService_GetPeriodDuration(t *testing.T) {
	logger := zerolog.Nop()
	service := NewTimeCalculatorService(&logger)

	period := &common.ScalerPeriod{
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				StartTime: "09:00",
				EndTime:   "17:00",
			},
		},
	}

	duration, err := service.GetPeriodDuration(period)

	assert.NoError(t, err)
	assert.Equal(t, 8*time.Hour, duration)
}

func TestTimeCalculatorService_CalculatePeriodStartTime(t *testing.T) {
	logger := zerolog.Nop()
	service := NewTimeCalculatorService(&logger)

	period := &common.ScalerPeriod{
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				StartTime: "09:00",
			},
		},
	}

	startTime, err := service.CalculatePeriodStartTime(period, 1*time.Hour)

	assert.NoError(t, err)
	assert.Equal(t, 10, startTime.Hour())
	assert.Equal(t, 0, startTime.Minute())
}
