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
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
)

func TestFlowProcessorService_ProcessFlow_Performance(t *testing.T) {
	logger := zerolog.Nop()

	// Create mocks
	mockValidator := &MockFlowValidator{}
	mockResourceMapper := &MockResourceMapper{}
	mockResourceCreator := &MockResourceCreator{}

	// Create service
	service := NewFlowProcessorService(mockValidator, mockResourceMapper, mockResourceCreator, &logger)

	// Create test flow
	flow := &kubecloudscalerv1alpha3.Flow{
		Spec: kubecloudscalerv1alpha3.FlowSpec{
			Flows: []kubecloudscalerv1alpha3.Flows{
				{
					PeriodName: "test-period",
					Resources: []kubecloudscalerv1alpha3.FlowResource{
						{Name: "test-resource"},
					},
				},
			},
		},
	}

	// Setup mocks
	mockValidator.ExtractFlowDataFunc = func(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error) {
		return map[string]bool{"test-resource": true}, map[string]bool{"test-period": true}, nil
	}
	mockValidator.ValidatePeriodTimingsFunc = func(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error {
		return nil
	}
	mockResourceMapper.CreateResourceMappingsFunc = func(flow *kubecloudscalerv1alpha3.Flow, resourceNames map[string]bool) (map[string]types.ResourceInfo, error) {
		return map[string]types.ResourceInfo{
			"test-resource": {
				Type:     "k8s",
				Resource: kubecloudscalerv1alpha3.K8sResource{Name: "test-resource"},
				Periods:  []types.PeriodWithDelay{},
			},
		}, nil
	}
	mockResourceCreator.CreateK8sResourceFunc = func(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, resourceName string, k8sResource kubecloudscalerv1alpha3.K8sResource, periodsWithDelay []types.PeriodWithDelay) error {
		return nil
	}

	// Measure performance
	start := time.Now()

	for i := 0; i < 100; i++ {
		err := service.ProcessFlow(context.Background(), flow)
		assert.NoError(t, err)
	}

	duration := time.Since(start)

	// Assert that processing 100 flows takes less than 1 second
	assert.Less(t, duration, time.Second)

	t.Logf("Processed 100 flows in %v (avg: %v per flow)", duration, duration/100)
}

func BenchmarkTimeCalculatorService_GetPeriodDuration(b *testing.B) {
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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetPeriodDuration(period)
	}
}

func BenchmarkTimeCalculatorService_CalculatePeriodStartTime(b *testing.B) {
	logger := zerolog.Nop()
	service := NewTimeCalculatorService(&logger)

	period := &common.ScalerPeriod{
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				StartTime: "09:00",
			},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.CalculatePeriodStartTime(period, time.Hour)
	}
}
