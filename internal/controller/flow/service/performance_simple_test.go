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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/testutil"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
)

func BenchmarkFlowProcessorService_ProcessFlow(b *testing.B) {
	logger := zerolog.Nop()

	mockValidator := &testutil.MockFlowValidator{}
	mockResourceMapper := &testutil.MockResourceMapper{}
	mockResourceCreator := &testutil.MockResourceCreator{}

	svc := NewFlowProcessorService(mockValidator, mockResourceMapper, mockResourceCreator, &logger)

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

	mockValidator.ExtractFlowDataFunc = func(
		flow *kubecloudscalerv1alpha3.Flow,
	) (map[string]bool, map[string]bool, error) {
		return map[string]bool{"test-resource": true}, map[string]bool{"test-period": true}, nil
	}
	mockValidator.ValidatePeriodTimingsFunc = func(
		flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool,
	) error {
		return nil
	}
	k8sRes := kubecloudscalerv1alpha3.K8sResource{Name: "test-resource"}
	mockResourceMapper.CreateResourceMappingsFunc = func(
		flow *kubecloudscalerv1alpha3.Flow, resourceNames map[string]bool,
	) (map[string]types.ResourceInfo, error) {
		return map[string]types.ResourceInfo{
			"test-resource": {
				Type:    "k8s",
				K8sRes:  &k8sRes,
				Periods: []types.PeriodWithDelay{},
			},
		}, nil
	}
	mockResourceCreator.CreateK8sResourceFunc = func(
		ctx context.Context,
		flow *kubecloudscalerv1alpha3.Flow,
		resourceName string,
		k8sResource kubecloudscalerv1alpha3.K8sResource,
		periodsWithDelay []types.PeriodWithDelay,
	) error {
		return nil
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := svc.ProcessFlow(context.Background(), flow)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTimeCalculatorService_GetPeriodDuration(b *testing.B) {
	logger := zerolog.Nop()
	svc := NewTimeCalculatorService(&logger)

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
		_, _ = svc.GetPeriodDuration(period)
	}
}

func BenchmarkTimeCalculatorService_CalculatePeriodStartTime(b *testing.B) {
	logger := zerolog.Nop()
	svc := NewTimeCalculatorService(&logger)

	period := &common.ScalerPeriod{
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				StartTime: "09:00",
			},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = svc.CalculatePeriodStartTime(period, time.Hour)
	}
}
