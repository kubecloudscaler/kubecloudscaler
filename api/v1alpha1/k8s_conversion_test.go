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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	v1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

func ptrString(s string) *string { return &s }
func ptrInt32(i int32) *int32    { return &i }
func ptrBool(b bool) *bool       { return &b }

func TestK8sConversion_ForwardRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  *K8s
	}{
		{
			name: "full spec round-trip",
			src: &K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-k8s",
				},
				Spec: K8sSpec{
					DryRun: true,
					Periods: []*common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday, common.DayFriday},
									StartTime: "08:00",
									EndTime:   "18:00",
									Timezone:  ptrString("Europe/Paris"),
									Once:      ptrBool(false),
								},
							},
							MinReplicas: ptrInt32(1),
							MaxReplicas: ptrInt32(5),
							Name:        "business-hours",
						},
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Fixed: &common.FixedPeriod{
									StartTime: "2025-01-01 00:00:00",
									EndTime:   "2025-01-02 00:00:00",
									Timezone:  ptrString("UTC"),
								},
							},
							Name: "maintenance-window",
						},
					},
					Namespaces:                   []string{"default", "production"},
					ExcludeNamespaces:            []string{"kube-system"},
					ForceExcludeSystemNamespaces: true,
					Resources:                    []string{"deployments", "statefulsets"},
					ExcludeResources:             []string{"my-deploy", "my-sts"},
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "web"},
					},
					DeploymentTimeAnnotation: "deploy.example.com/time",
					DisableEvents:            true,
					AuthSecret:               ptrString("my-secret"),
					RestoreOnDelete:          true,
				},
				Status: common.ScalerStatus{
					Comments: ptrString("test status"),
				},
			},
		},
		{
			name: "minimal spec",
			src: &K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name: "minimal",
				},
				Spec: K8sSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayAll},
									StartTime: "00:00",
									EndTime:   "23:59",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// v1alpha1 -> v1alpha3
			hub := &v1alpha3.K8s{}
			require.NoError(t, tt.src.ConvertTo(hub))

			// v1alpha3 -> v1alpha1
			result := &K8s{}
			require.NoError(t, result.ConvertFrom(hub))

			assert.Equal(t, tt.src.Name, result.Name)
			assert.Equal(t, tt.src.Spec.DryRun, result.Spec.DryRun)
			assert.Equal(t, tt.src.Spec.Namespaces, result.Spec.Namespaces)
			assert.Equal(t, tt.src.Spec.ExcludeNamespaces, result.Spec.ExcludeNamespaces)
			assert.Equal(t, tt.src.Spec.ForceExcludeSystemNamespaces, result.Spec.ForceExcludeSystemNamespaces)
			assert.Equal(t, tt.src.Spec.Resources, result.Spec.Resources)
			assert.Equal(t, tt.src.Spec.DeploymentTimeAnnotation, result.Spec.DeploymentTimeAnnotation)
			assert.Equal(t, tt.src.Spec.DisableEvents, result.Spec.DisableEvents)
			assert.Equal(t, tt.src.Spec.AuthSecret, result.Spec.AuthSecret)
			assert.Equal(t, tt.src.Spec.RestoreOnDelete, result.Spec.RestoreOnDelete)
			assert.Equal(t, tt.src.Spec.LabelSelector, result.Spec.LabelSelector)
			assert.Equal(t, tt.src.Status, result.Status)

			// Period count must match (nil pointers are filtered)
			expectedPeriods := 0
			for _, p := range tt.src.Spec.Periods {
				if p != nil {
					expectedPeriods++
				}
			}
			require.Len(t, result.Spec.Periods, expectedPeriods)
			for i, p := range result.Spec.Periods {
				require.NotNil(t, p)
				assert.Equal(t, *tt.src.Spec.Periods[i], *p)
			}
		})
	}
}

func TestK8sConversion_BackwardRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  *v1alpha3.K8s
	}{
		{
			name: "full v1alpha3 spec round-trip",
			src: &v1alpha3.K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-k8s-v3",
				},
				Spec: v1alpha3.K8sSpec{
					DryRun: true,
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday, common.DayTuesday},
									StartTime: "09:00",
									EndTime:   "17:00",
									Timezone:  ptrString("America/New_York"),
								},
							},
							MinReplicas: ptrInt32(2),
							MaxReplicas: ptrInt32(10),
							Name:        "work-hours",
						},
					},
					Resources: common.Resources{
						Types:         []common.ResourceKind{common.ResourceDeployments, common.ResourceStatefulSets},
						LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"tier": "backend"}},
					},
					Config: v1alpha3.K8sConfig{
						Namespaces:                   []string{"app-ns"},
						ExcludeNamespaces:            []string{"monitoring"},
						ForceExcludeSystemNamespaces: true,
						DeploymentTimeAnnotation:     "deploy-at",
						DisableEvents:                false,
						AuthSecret:                   ptrString("cluster-secret"),
						RestoreOnDelete:              true,
					},
				},
				Status: common.ScalerStatus{
					Comments: ptrString("active"),
				},
			},
		},
		{
			name: "empty slices round-trip",
			src: &v1alpha3.K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-slices",
				},
				Spec: v1alpha3.K8sSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DaySunday},
									StartTime: "00:00",
									EndTime:   "06:00",
								},
							},
						},
					},
					Resources: common.Resources{},
					Config:    v1alpha3.K8sConfig{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// v1alpha3 -> v1alpha1
			intermediate := &K8s{}
			require.NoError(t, intermediate.ConvertFrom(tt.src))

			// v1alpha1 -> v1alpha3
			result := &v1alpha3.K8s{}
			require.NoError(t, intermediate.ConvertTo(result))

			assert.Equal(t, tt.src.Name, result.Name)
			assert.Equal(t, tt.src.Spec.DryRun, result.Spec.DryRun)
			assert.Equal(t, tt.src.Spec.Config.Namespaces, result.Spec.Config.Namespaces)
			assert.Equal(t, tt.src.Spec.Config.ExcludeNamespaces, result.Spec.Config.ExcludeNamespaces)
			assert.Equal(t, tt.src.Spec.Config.ForceExcludeSystemNamespaces, result.Spec.Config.ForceExcludeSystemNamespaces)
			assert.Equal(t, tt.src.Spec.Config.DeploymentTimeAnnotation, result.Spec.Config.DeploymentTimeAnnotation)
			assert.Equal(t, tt.src.Spec.Config.DisableEvents, result.Spec.Config.DisableEvents)
			assert.Equal(t, tt.src.Spec.Config.AuthSecret, result.Spec.Config.AuthSecret)
			assert.Equal(t, tt.src.Spec.Config.RestoreOnDelete, result.Spec.Config.RestoreOnDelete)
			assert.Equal(t, tt.src.Spec.Resources.Types, result.Spec.Resources.Types)
			assert.Equal(t, tt.src.Spec.Resources.LabelSelector, result.Spec.Resources.LabelSelector)
			assert.Equal(t, tt.src.Status, result.Status)

			require.Len(t, result.Spec.Periods, len(tt.src.Spec.Periods))
			for i := range tt.src.Spec.Periods {
				assert.Equal(t, tt.src.Spec.Periods[i], result.Spec.Periods[i])
			}
		})
	}
}

func TestK8sConversion_AnnotationPreservation(t *testing.T) {
	src := &K8s{
		ObjectMeta: metav1.ObjectMeta{
			Name: "annotation-test",
		},
		Spec: K8sSpec{
			Periods: []*common.ScalerPeriod{
				{
					Type: common.PeriodTypeDown,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayAll},
							StartTime: "00:00",
							EndTime:   "23:59",
						},
					},
				},
			},
			ExcludeResources: []string{"deploy/my-app", "sts/my-db"},
		},
	}

	hub := &v1alpha3.K8s{}
	require.NoError(t, src.ConvertTo(hub))

	// Verify annotation was set on hub
	assert.Contains(t, hub.Annotations, annotationExcludeResources)

	// Convert back and verify ExcludeResources survived
	result := &K8s{}
	require.NoError(t, result.ConvertFrom(hub))
	assert.Equal(t, src.Spec.ExcludeResources, result.Spec.ExcludeResources)

	// Annotations should be cleaned up after ConvertFrom
	assert.Nil(t, result.Annotations)
}

func TestK8sConversion_NilPeriodHandling(t *testing.T) {
	src := &K8s{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nil-periods",
		},
		Spec: K8sSpec{
			Periods: []*common.ScalerPeriod{
				{
					Type: common.PeriodTypeDown,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayMonday},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
				},
				nil,
				{
					Type: common.PeriodTypeUp,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DaySaturday},
							StartTime: "10:00",
							EndTime:   "14:00",
						},
					},
				},
			},
		},
	}

	hub := &v1alpha3.K8s{}
	require.NoError(t, src.ConvertTo(hub))

	// Nil period should be filtered out
	require.Len(t, hub.Spec.Periods, 2)
	assert.Equal(t, common.PeriodTypeDown, hub.Spec.Periods[0].Type)
	assert.Equal(t, common.PeriodTypeUp, hub.Spec.Periods[1].Type)

	// Round-trip back
	result := &K8s{}
	require.NoError(t, result.ConvertFrom(hub))
	require.Len(t, result.Spec.Periods, 2)
	for _, p := range result.Spec.Periods {
		require.NotNil(t, p)
	}
}

func TestK8sConversion_EmptyNilSlices(t *testing.T) {
	tests := []struct {
		name       string
		resources  []string
		namespaces []string
	}{
		{
			name:       "nil resources and namespaces",
			resources:  nil,
			namespaces: nil,
		},
		{
			name:       "empty resources and namespaces",
			resources:  []string{},
			namespaces: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-test",
				},
				Spec: K8sSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayAll},
									StartTime: "00:00",
									EndTime:   "23:59",
								},
							},
						},
					},
					Resources:  tt.resources,
					Namespaces: tt.namespaces,
				},
			}

			hub := &v1alpha3.K8s{}
			require.NoError(t, src.ConvertTo(hub))

			result := &K8s{}
			require.NoError(t, result.ConvertFrom(hub))

			// Nil and empty slices may not be identical, but both should be empty
			assert.Empty(t, result.Spec.Resources)
			assert.Empty(t, result.Spec.Namespaces)
		})
	}
}

func TestK8sConversion_TypeAssertionError(t *testing.T) {
	src := &K8s{}
	wrongHub := &v1alpha3.Gcp{}

	err := src.ConvertTo(wrongHub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *kubecloudscalercloudv1alpha3.K8s")

	err = src.ConvertFrom(wrongHub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *kubecloudscalercloudv1alpha3.K8s")
}
