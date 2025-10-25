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

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

func TestStatusUpdaterService_UpdateFlowStatus_Simple(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kubecloudscalerv1alpha3.AddToScheme(scheme)

	logger := zerolog.Nop()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&kubecloudscalerv1alpha3.Flow{}).
		Build()

	service := NewStatusUpdaterService(fakeClient, &logger)

	t.Run("successful status update", func(t *testing.T) {
		flow := &kubecloudscalerv1alpha3.Flow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-flow",
				Namespace: "default",
			},
			Status: kubecloudscalerv1alpha3.FlowStatus{
				Conditions: []metav1.Condition{},
			},
		}

		// Create the flow in the fake client
		err := fakeClient.Create(context.Background(), flow)
		assert.NoError(t, err)

		condition := metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionTrue,
			Reason:  "ProcessingSucceeded",
			Message: "Flow processed successfully",
		}

		result, err := service.UpdateFlowStatus(context.Background(), flow, condition)

		assert.NoError(t, err)
		assert.Equal(t, utils.ReconcileSuccessDuration, result.RequeueAfter)
	})

	t.Run("error condition", func(t *testing.T) {
		flow := &kubecloudscalerv1alpha3.Flow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-flow-error",
				Namespace: "default",
			},
			Status: kubecloudscalerv1alpha3.FlowStatus{
				Conditions: []metav1.Condition{},
			},
		}

		// Create the flow in the fake client
		err := fakeClient.Create(context.Background(), flow)
		assert.NoError(t, err)

		condition := metav1.Condition{
			Type:    "Error",
			Status:  metav1.ConditionTrue,
			Reason:  "ProcessingFailed",
			Message: "Flow processing failed",
		}

		result, err := service.UpdateFlowStatus(context.Background(), flow, condition)

		assert.NoError(t, err)
		assert.Equal(t, utils.ReconcileSuccessDuration, result.RequeueAfter)
	})
}
