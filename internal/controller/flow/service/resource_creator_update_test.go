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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

func TestResourceCreatorService_createOrUpdateResource(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kubecloudscalerv1alpha3.AddToScheme(scheme)

	logger := zerolog.Nop()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&kubecloudscalerv1alpha3.K8s{}).
		Build()

	service := NewResourceCreatorService(fakeClient, scheme, &logger)

	t.Run("create new resource", func(t *testing.T) {
		obj := &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-k8s",
				Namespace: "default",
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				DryRun: false,
			},
		}

		err := service.createOrUpdateResource(context.Background(), obj)

		assert.NoError(t, err)

		// Verify the resource was created
		var createdObj kubecloudscalerv1alpha3.K8s
		err = fakeClient.Get(context.Background(), types.NamespacedName{
			Name:      "test-k8s",
			Namespace: "default",
		}, &createdObj)

		assert.NoError(t, err)
		assert.Equal(t, "test-k8s", createdObj.Name)
	})

	t.Run("update existing resource", func(t *testing.T) {
		// Create initial resource
		initialObj := &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-k8s-update",
				Namespace: "default",
				Labels: map[string]string{
					"original": "label",
				},
				Annotations: map[string]string{
					"original": "annotation",
				},
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				DryRun: false,
			},
		}

		err := fakeClient.Create(context.Background(), initialObj)
		assert.NoError(t, err)

		// Update the resource with new spec
		updatedObj := &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-k8s-update",
				Namespace: "default",
				Labels: map[string]string{
					"new": "label",
				},
				Annotations: map[string]string{
					"new": "annotation",
				},
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				DryRun: true, // Changed from false to true
			},
		}

		err = service.createOrUpdateResource(context.Background(), updatedObj)
		assert.NoError(t, err)

		// Verify the resource was updated
		var finalObj kubecloudscalerv1alpha3.K8s
		err = fakeClient.Get(context.Background(), types.NamespacedName{
			Name:      "test-k8s-update",
			Namespace: "default",
		}, &finalObj)

		assert.NoError(t, err)
		assert.True(t, finalObj.Spec.DryRun)                            // Should be updated to true
		assert.Equal(t, "label", finalObj.Labels["original"])           // Original labels should be preserved
		assert.Equal(t, "label", finalObj.Labels["new"])                // New labels should be added
		assert.Equal(t, "annotation", finalObj.Annotations["original"]) // Original annotations should be preserved
		assert.Equal(t, "annotation", finalObj.Annotations["new"])      // New annotations should be added
	})

	t.Run("update with new labels and annotations", func(t *testing.T) {
		// Create initial resource
		initialObj := &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-k8s-labels",
				Namespace: "default",
				Labels: map[string]string{
					"existing": "label",
				},
				Annotations: map[string]string{
					"existing": "annotation",
				},
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				DryRun: false,
			},
		}

		err := fakeClient.Create(context.Background(), initialObj)
		assert.NoError(t, err)

		// Update with new labels and annotations
		updatedObj := &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-k8s-labels",
				Namespace: "default",
				Labels: map[string]string{
					"existing": "label",
					"new":      "label",
				},
				Annotations: map[string]string{
					"existing": "annotation",
					"new":      "annotation",
				},
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				DryRun: true,
			},
		}

		err = service.createOrUpdateResource(context.Background(), updatedObj)
		assert.NoError(t, err)

		// Verify the resource was updated with preserved and new metadata
		var finalObj kubecloudscalerv1alpha3.K8s
		err = fakeClient.Get(context.Background(), types.NamespacedName{
			Name:      "test-k8s-labels",
			Namespace: "default",
		}, &finalObj)

		assert.NoError(t, err)
		assert.True(t, finalObj.Spec.DryRun)
		assert.Equal(t, "label", finalObj.Labels["existing"])
		assert.Equal(t, "label", finalObj.Labels["new"])
		assert.Equal(t, "annotation", finalObj.Annotations["existing"])
		assert.Equal(t, "annotation", finalObj.Annotations["new"])
	})
}
