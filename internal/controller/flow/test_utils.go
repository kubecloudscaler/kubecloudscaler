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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// TestFlowBuilder provides a fluent interface for building test flows
type TestFlowBuilder struct {
	flow *kubecloudscalerv1alpha3.Flow
}

// NewTestFlowBuilder creates a new TestFlowBuilder
func NewTestFlowBuilder(name, namespace string) *TestFlowBuilder {
	return &TestFlowBuilder{
		flow: &kubecloudscalerv1alpha3.Flow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: kubecloudscalerv1alpha3.FlowSpec{
				Resources: kubecloudscalerv1alpha3.Resources{},
			},
		},
	}
}

// WithPeriod adds a period to the flow
func (b *TestFlowBuilder) WithPeriod(periodType, name string, startTime, endTime string) *TestFlowBuilder {
	period := common.ScalerPeriod{
		Type: periodType,
		Name: ptr.To(name),
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				Days:      []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
				StartTime: startTime,
				EndTime:   endTime,
			},
		},
	}
	b.flow.Spec.Periods = append(b.flow.Spec.Periods, period)
	return b
}

// WithFixedPeriod adds a fixed period to the flow
func (b *TestFlowBuilder) WithFixedPeriod(periodType, name, startTime, endTime string) *TestFlowBuilder {
	period := common.ScalerPeriod{
		Type: periodType,
		Name: ptr.To(name),
		Time: common.TimePeriod{
			Fixed: &common.FixedPeriod{
				StartTime: startTime,
				EndTime:   endTime,
			},
		},
	}
	b.flow.Spec.Periods = append(b.flow.Spec.Periods, period)
	return b
}

// WithK8sResource adds a K8s resource to the flow
func (b *TestFlowBuilder) WithK8sResource(name string, resourceTypes, resourceNames, namespaces []string) *TestFlowBuilder {
	k8sResource := kubecloudscalerv1alpha3.K8sResource{
		Name: name,
		Resources: common.Resources{
			Types: resourceTypes,
			Names: resourceNames,
		},
		Config: kubecloudscalerv1alpha3.K8sConfig{
			Namespaces: namespaces,
		},
	}
	b.flow.Spec.Resources.K8s = append(b.flow.Spec.Resources.K8s, k8sResource)
	return b
}

// WithGcpResource adds a GCP resource to the flow
func (b *TestFlowBuilder) WithGcpResource(name string, resourceTypes, resourceNames []string) *TestFlowBuilder {
	gcpResource := kubecloudscalerv1alpha3.GcpResource{
		Name: name,
		Resources: common.Resources{
			Types: resourceTypes,
			Names: resourceNames,
		},
	}
	b.flow.Spec.Resources.Gcp = append(b.flow.Spec.Resources.Gcp, gcpResource)
	return b
}

// WithFlow adds a flow configuration to the flow
func (b *TestFlowBuilder) WithFlow(periodName string, resources []FlowResourceConfig) *TestFlowBuilder {
	flowResources := make([]kubecloudscalerv1alpha3.FlowResource, len(resources))
	for i, res := range resources {
		flowResources[i] = kubecloudscalerv1alpha3.FlowResource{
			Name:           res.Name,
			StartTimeDelay: res.StartTimeDelay,
			EndTimeDelay:   res.EndTimeDelay,
		}
	}

	flow := kubecloudscalerv1alpha3.Flows{
		PeriodName: periodName,
		Resources:  flowResources,
	}
	b.flow.Spec.Flows = append(b.flow.Spec.Flows, flow)
	return b
}

// WithFinalizer adds a finalizer to the flow
func (b *TestFlowBuilder) WithFinalizer(finalizer string) *TestFlowBuilder {
	b.flow.Finalizers = append(b.flow.Finalizers, finalizer)
	return b
}

// WithDeletionTimestamp sets the deletion timestamp
func (b *TestFlowBuilder) WithDeletionTimestamp() *TestFlowBuilder {
	now := metav1.Now()
	b.flow.DeletionTimestamp = &now
	return b
}

// Build returns the constructed flow
func (b *TestFlowBuilder) Build() *kubecloudscalerv1alpha3.Flow {
	return b.flow
}

// FlowResourceConfig represents configuration for a flow resource
type FlowResourceConfig struct {
	Name           string
	StartTimeDelay string
	EndTimeDelay   string
}

// CreateValidFlow creates a valid test flow
func CreateValidFlow(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "test-period", "09:00", "17:00").
		WithK8sResource("test-k8s", []string{"deployments"}, []string{"test-deployment"}, []string{"default"}).
		WithFlow("test-period", []FlowResourceConfig{
			{Name: "test-k8s", StartTimeDelay: "10m"},
		}).
		Build()
}

// CreateFlowWithGcpResource creates a flow with GCP resource
func CreateFlowWithGcpResource(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "test-period", "09:00", "17:00").
		WithGcpResource("test-gcp", []string{"vm-instances"}, []string{"test-instance"}).
		WithFlow("test-period", []FlowResourceConfig{
			{Name: "test-gcp", StartTimeDelay: "15m"},
		}).
		Build()
}

// CreateFlowWithDelays creates a flow with timing delays
func CreateFlowWithDelays(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "delayed-period", "09:00", "17:00").
		WithK8sResource("delayed-k8s", []string{"deployments"}, []string{"test-deployment"}, []string{"default"}).
		WithFlow("delayed-period", []FlowResourceConfig{
			{Name: "delayed-k8s", StartTimeDelay: "30m", EndTimeDelay: "15m"},
		}).
		Build()
}

// CreateInvalidFlow creates an invalid flow for error testing
func CreateInvalidFlow(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "valid-period", "09:00", "17:00").
		WithK8sResource("test-k8s", []string{"deployments"}, []string{"test-deployment"}, []string{"default"}).
		WithFlow("invalid-period", []FlowResourceConfig{ // Invalid period name
			{Name: "test-k8s"},
		}).
		Build()
}

// CreateFlowWithInvalidResource creates a flow with invalid resource reference
func CreateFlowWithInvalidResource(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "test-period", "09:00", "17:00").
		WithK8sResource("valid-k8s", []string{"deployments"}, []string{"test-deployment"}, []string{"default"}).
		WithFlow("test-period", []FlowResourceConfig{
			{Name: "invalid-resource"}, // Invalid resource name
		}).
		Build()
}

// CreateFlowWithInvalidDelay creates a flow with invalid delay format
func CreateFlowWithInvalidDelay(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "test-period", "09:00", "17:00").
		WithK8sResource("test-k8s", []string{"deployments"}, []string{"test-deployment"}, []string{"default"}).
		WithFlow("test-period", []FlowResourceConfig{
			{Name: "test-k8s", StartTimeDelay: "invalid-delay"},
		}).
		Build()
}

// CreateFlowForDeletion creates a flow ready for deletion
func CreateFlowForDeletion(name, namespace string) *kubecloudscalerv1alpha3.Flow {
	return NewTestFlowBuilder(name, namespace).
		WithPeriod("up", "test-period", "09:00", "17:00").
		WithK8sResource("test-k8s", []string{"deployments"}, []string{"test-deployment"}, []string{"default"}).
		WithFlow("test-period", []FlowResourceConfig{
			{Name: "test-k8s"},
		}).
		WithFinalizer("kubecloudscaler.cloud/flow-finalizer").
		WithDeletionTimestamp().
		Build()
}

// CreateLargeFlow creates a flow with many periods and resources for performance testing
func CreateLargeFlow(name, namespace string, numPeriods, resourcesPerPeriod int) *kubecloudscalerv1alpha3.Flow {
	builder := NewTestFlowBuilder(name, namespace)

	// Add periods
	for i := 0; i < numPeriods; i++ {
		startHour := 9 + (i % 8)
		endHour := startHour + 8
		builder.WithPeriod("up", fmt.Sprintf("period-%d", i),
			fmt.Sprintf("%02d:00", startHour),
			fmt.Sprintf("%02d:00", endHour))
	}

	// Add K8s resources
	for i := 0; i < numPeriods*resourcesPerPeriod; i++ {
		builder.WithK8sResource(
			fmt.Sprintf("resource-%d", i),
			[]string{"deployments"},
			[]string{fmt.Sprintf("deployment-%d", i)},
			[]string{"default"},
		)
	}

	// Add flows
	for i := 0; i < numPeriods; i++ {
		resources := make([]FlowResourceConfig, resourcesPerPeriod)
		for j := 0; j < resourcesPerPeriod; j++ {
			resourceIndex := i*resourcesPerPeriod + j
			resources[j] = FlowResourceConfig{
				Name:           fmt.Sprintf("resource-%d", resourceIndex),
				StartTimeDelay: "10m",
				EndTimeDelay:   "5m",
			}
		}
		builder.WithFlow(fmt.Sprintf("period-%d", i), resources)
	}

	return builder.Build()
}

// AssertFlowStatus checks if the flow has the expected status condition
func AssertFlowStatus(flow *kubecloudscalerv1alpha3.Flow, conditionType string, status metav1.ConditionStatus) error {
	if len(flow.Status.Conditions) == 0 {
		return fmt.Errorf("expected at least one condition, got none")
	}

	lastCondition := flow.Status.Conditions[len(flow.Status.Conditions)-1]
	if lastCondition.Type != conditionType {
		return fmt.Errorf("expected condition type %s, got %s", conditionType, lastCondition.Type)
	}

	if lastCondition.Status != status {
		return fmt.Errorf("expected condition status %s, got %s", status, lastCondition.Status)
	}

	return nil
}

// AssertK8sResourceCreated checks if a K8s resource was created with expected properties
func AssertK8sResourceCreated(client client.Client, ctx context.Context, expectedName, expectedNamespace string, expectedPeriods int) error {
	k8sResource := &kubecloudscalerv1alpha3.K8s{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      expectedName,
		Namespace: expectedNamespace,
	}, k8sResource)

	if err != nil {
		return fmt.Errorf("failed to get K8s resource: %w", err)
	}

	if len(k8sResource.Spec.Periods) != expectedPeriods {
		return fmt.Errorf("expected %d periods, got %d", expectedPeriods, len(k8sResource.Spec.Periods))
	}

	return nil
}

// AssertGcpResourceCreated checks if a GCP resource was created with expected properties
func AssertGcpResourceCreated(client client.Client, ctx context.Context, expectedName, expectedNamespace string, expectedPeriods int) error {
	gcpResource := &kubecloudscalerv1alpha3.Gcp{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      expectedName,
		Namespace: expectedNamespace,
	}, gcpResource)

	if err != nil {
		return fmt.Errorf("failed to get GCP resource: %w", err)
	}

	if len(gcpResource.Spec.Periods) != expectedPeriods {
		return fmt.Errorf("expected %d periods, got %d", expectedPeriods, len(gcpResource.Spec.Periods))
	}

	return nil
}
