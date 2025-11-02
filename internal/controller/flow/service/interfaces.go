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

// Package service provides interfaces for the flow service.
package service

import (
	"context"
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// FlowProcessor defines the interface for processing flow resources
type FlowProcessor interface {
	ProcessFlow(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow) error
}

// ResourceCreator defines the interface for creating K8s and GCP resources
type ResourceCreator interface {
	CreateK8sResource(
		ctx context.Context,
		flow *kubecloudscalerv1alpha3.Flow,
		resourceName string,
		k8sResource kubecloudscalerv1alpha3.K8sResource,
		periodsWithDelay []types.PeriodWithDelay,
	) error
	CreateGcpResource(
		ctx context.Context,
		flow *kubecloudscalerv1alpha3.Flow,
		resourceName string,
		gcpResource kubecloudscalerv1alpha3.GcpResource,
		periodsWithDelay []types.PeriodWithDelay,
	) error
}

// FlowValidator defines the interface for validating flow configurations
type FlowValidator interface {
	ValidatePeriodTimings(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error
	ExtractFlowData(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error)
}

// TimeCalculator defines the interface for time-related calculations
type TimeCalculator interface {
	CalculatePeriodStartTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error)
	CalculatePeriodEndTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error)
	GetPeriodDuration(period *common.ScalerPeriod) (time.Duration, error)
}

// ResourceMapper defines the interface for mapping resources with periods
type ResourceMapper interface {
	CreateResourceMappings(
		flow *kubecloudscalerv1alpha3.Flow,
		resourceNames map[string]bool,
	) (map[string]types.ResourceInfo, error)
}

// StatusUpdater defines the interface for updating flow status
type StatusUpdater interface {
	UpdateFlowStatus(
		ctx context.Context,
		flow *kubecloudscalerv1alpha3.Flow,
		condition metav1.Condition,
	) (ctrl.Result, error)
}
