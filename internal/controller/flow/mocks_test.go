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

package controller

import (
	"context"
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// MockFlowProcessor is a mock implementation of FlowProcessor
type MockFlowProcessor struct {
	ProcessFlowFunc func(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow) error
}

func (m *MockFlowProcessor) ProcessFlow(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow) error {
	if m.ProcessFlowFunc != nil {
		return m.ProcessFlowFunc(ctx, flow)
	}
	return nil
}

// MockResourceCreator is a mock implementation of ResourceCreator
type MockResourceCreator struct {
	CreateK8sResourceFunc func(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, resourceName string, k8sResource kubecloudscalerv1alpha3.K8sResource, periodsWithDelay []PeriodWithDelay) error
	CreateGcpResourceFunc func(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, resourceName string, gcpResource kubecloudscalerv1alpha3.GcpResource, periodsWithDelay []PeriodWithDelay) error
}

func (m *MockResourceCreator) CreateK8sResource(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, resourceName string, k8sResource kubecloudscalerv1alpha3.K8sResource, periodsWithDelay []PeriodWithDelay) error {
	if m.CreateK8sResourceFunc != nil {
		return m.CreateK8sResourceFunc(ctx, flow, resourceName, k8sResource, periodsWithDelay)
	}
	return nil
}

func (m *MockResourceCreator) CreateGcpResource(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, resourceName string, gcpResource kubecloudscalerv1alpha3.GcpResource, periodsWithDelay []PeriodWithDelay) error {
	if m.CreateGcpResourceFunc != nil {
		return m.CreateGcpResourceFunc(ctx, flow, resourceName, gcpResource, periodsWithDelay)
	}
	return nil
}

// MockFlowValidator is a mock implementation of FlowValidator
type MockFlowValidator struct {
	ValidatePeriodTimingsFunc func(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error
	ExtractFlowDataFunc       func(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error)
}

func (m *MockFlowValidator) ValidatePeriodTimings(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error {
	if m.ValidatePeriodTimingsFunc != nil {
		return m.ValidatePeriodTimingsFunc(flow, periodNames)
	}
	return nil
}

func (m *MockFlowValidator) ExtractFlowData(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error) {
	if m.ExtractFlowDataFunc != nil {
		return m.ExtractFlowDataFunc(flow)
	}
	return make(map[string]bool), make(map[string]bool), nil
}

// MockTimeCalculator is a mock implementation of TimeCalculator
type MockTimeCalculator struct {
	CalculatePeriodStartTimeFunc func(period *common.ScalerPeriod, delay time.Duration) (time.Time, error)
	CalculatePeriodEndTimeFunc   func(period *common.ScalerPeriod, delay time.Duration) (time.Time, error)
	GetPeriodDurationFunc        func(period *common.ScalerPeriod) (time.Duration, error)
}

func (m *MockTimeCalculator) CalculatePeriodStartTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error) {
	if m.CalculatePeriodStartTimeFunc != nil {
		return m.CalculatePeriodStartTimeFunc(period, delay)
	}
	return time.Now(), nil
}

func (m *MockTimeCalculator) CalculatePeriodEndTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error) {
	if m.CalculatePeriodEndTimeFunc != nil {
		return m.CalculatePeriodEndTimeFunc(period, delay)
	}
	return time.Now().Add(8 * time.Hour), nil
}

func (m *MockTimeCalculator) GetPeriodDuration(period *common.ScalerPeriod) (time.Duration, error) {
	if m.GetPeriodDurationFunc != nil {
		return m.GetPeriodDurationFunc(period)
	}
	return 8 * time.Hour, nil
}

// MockStatusUpdater is a mock implementation of StatusUpdater
type MockStatusUpdater struct {
	UpdateFlowStatusFunc func(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, condition metav1.Condition) (ctrl.Result, error)
}

func (m *MockStatusUpdater) UpdateFlowStatus(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, condition metav1.Condition) (ctrl.Result, error) {
	if m.UpdateFlowStatusFunc != nil {
		return m.UpdateFlowStatusFunc(ctx, flow, condition)
	}
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}
