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

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	v1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

func TestGcpConversion_ForwardRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  *Gcp
	}{
		{
			name: "full spec round-trip",
			src: &Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gcp-v2",
				},
				Spec: GcpSpec{
					DryRun: true,
					Periods: []*common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday, common.DayFriday},
									StartTime: "20:00",
									EndTime:   "06:00",
									Timezone:  ptrString("Europe/Paris"),
								},
							},
							Name: "night-shutdown",
						},
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Fixed: &common.FixedPeriod{
									StartTime: "2025-06-01 00:00:00",
									EndTime:   "2025-06-02 00:00:00",
								},
							},
							Name: "maintenance",
						},
					},
					Resources: common.Resources{
						Types:         []common.ResourceKind{common.ResourceVMInstances},
						LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"env": "staging"}},
						Names:         []string{"worker-1", "worker-2"},
					},
					ProjectID:         "my-project-123",
					Region:            "europe-west1",
					AuthSecret:        ptrString("gcp-secret"),
					RestoreOnDelete:   true,
					WaitForOperation:  true,
					DefaultPeriodType: "up",
				},
				Status: common.ScalerStatus{
					Comments: ptrString("running"),
				},
			},
		},
		{
			name: "minimal spec",
			src: &Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name: "minimal-gcp-v2",
				},
				Spec: GcpSpec{
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
					ProjectID:         "proj",
					DefaultPeriodType: "down",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// v1alpha2 -> v1alpha3
			hub := &v1alpha3.Gcp{}
			require.NoError(t, tt.src.ConvertTo(hub))

			// v1alpha3 -> v1alpha2
			result := &Gcp{}
			require.NoError(t, result.ConvertFrom(hub))

			assert.Equal(t, tt.src.Name, result.Name)
			assert.Equal(t, tt.src.Spec.DryRun, result.Spec.DryRun)
			assert.Equal(t, tt.src.Spec.ProjectID, result.Spec.ProjectID)
			assert.Equal(t, tt.src.Spec.Region, result.Spec.Region)
			assert.Equal(t, tt.src.Spec.AuthSecret, result.Spec.AuthSecret)
			assert.Equal(t, tt.src.Spec.RestoreOnDelete, result.Spec.RestoreOnDelete)
			assert.Equal(t, tt.src.Spec.WaitForOperation, result.Spec.WaitForOperation)
			assert.Equal(t, tt.src.Spec.DefaultPeriodType, result.Spec.DefaultPeriodType)
			assert.Equal(t, tt.src.Spec.Resources, result.Spec.Resources)
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

func TestGcpConversion_BackwardRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  *v1alpha3.Gcp
	}{
		{
			name: "full v1alpha3 spec round-trip",
			src: &v1alpha3.Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gcp-v3",
				},
				Spec: v1alpha3.GcpSpec{
					DryRun: true,
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday},
									StartTime: "07:00",
									EndTime:   "19:00",
									Timezone:  ptrString("UTC"),
								},
							},
							MinReplicas: ptrInt32(1),
							Name:        "daytime",
						},
					},
					Resources: common.Resources{
						Types:         []common.ResourceKind{common.ResourceVMInstances},
						LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"pool": "workers"}},
					},
					Config: v1alpha3.GcpConfig{
						ProjectID:         "prod-project",
						Region:            "us-central1",
						AuthSecret:        ptrString("sa-key"),
						RestoreOnDelete:   true,
						WaitForOperation:  true,
						DefaultPeriodType: "up",
					},
				},
				Status: common.ScalerStatus{
					Comments: ptrString("healthy"),
				},
			},
		},
		{
			name: "empty resources round-trip",
			src: &v1alpha3.Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-res",
				},
				Spec: v1alpha3.GcpSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DaySaturday},
									StartTime: "00:00",
									EndTime:   "23:59",
								},
							},
						},
					},
					Resources: common.Resources{},
					Config: v1alpha3.GcpConfig{
						ProjectID:         "test",
						DefaultPeriodType: "down",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// v1alpha3 -> v1alpha2
			intermediate := &Gcp{}
			require.NoError(t, intermediate.ConvertFrom(tt.src))

			// v1alpha2 -> v1alpha3
			result := &v1alpha3.Gcp{}
			require.NoError(t, intermediate.ConvertTo(result))

			assert.Equal(t, tt.src.Name, result.Name)
			assert.Equal(t, tt.src.Spec.DryRun, result.Spec.DryRun)
			assert.Equal(t, tt.src.Spec.Config.ProjectID, result.Spec.Config.ProjectID)
			assert.Equal(t, tt.src.Spec.Config.Region, result.Spec.Config.Region)
			assert.Equal(t, tt.src.Spec.Config.AuthSecret, result.Spec.Config.AuthSecret)
			assert.Equal(t, tt.src.Spec.Config.RestoreOnDelete, result.Spec.Config.RestoreOnDelete)
			assert.Equal(t, tt.src.Spec.Config.WaitForOperation, result.Spec.Config.WaitForOperation)
			assert.Equal(t, tt.src.Spec.Config.DefaultPeriodType, result.Spec.Config.DefaultPeriodType)
			assert.Equal(t, tt.src.Spec.Resources, result.Spec.Resources)
			assert.Equal(t, tt.src.Status, result.Status)

			require.Len(t, result.Spec.Periods, len(tt.src.Spec.Periods))
			for i := range tt.src.Spec.Periods {
				assert.Equal(t, tt.src.Spec.Periods[i], result.Spec.Periods[i])
			}
		})
	}
}

func TestGcpConversion_NilPeriodHandling(t *testing.T) {
	src := &Gcp{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nil-periods",
		},
		Spec: GcpSpec{
			Periods: []*common.ScalerPeriod{
				{
					Type: common.PeriodTypeDown,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayMonday},
							StartTime: "22:00",
							EndTime:   "06:00",
						},
					},
				},
				nil,
				{
					Type: common.PeriodTypeUp,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayWednesday},
							StartTime: "08:00",
							EndTime:   "20:00",
						},
					},
				},
			},
			ProjectID:         "test",
			DefaultPeriodType: "down",
		},
	}

	hub := &v1alpha3.Gcp{}
	require.NoError(t, src.ConvertTo(hub))

	// Nil periods should be filtered
	require.Len(t, hub.Spec.Periods, 2)
	assert.Equal(t, common.PeriodTypeDown, hub.Spec.Periods[0].Type)
	assert.Equal(t, common.PeriodTypeUp, hub.Spec.Periods[1].Type)

	result := &Gcp{}
	require.NoError(t, result.ConvertFrom(hub))
	require.Len(t, result.Spec.Periods, 2)
	for _, p := range result.Spec.Periods {
		require.NotNil(t, p)
	}
}

func TestGcpConversion_EmptyNilSlices(t *testing.T) {
	tests := []struct {
		name      string
		resources common.Resources
	}{
		{name: "nil types", resources: common.Resources{Types: nil}},
		{name: "empty types", resources: common.Resources{Types: []common.ResourceKind{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-test",
				},
				Spec: GcpSpec{
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
					ProjectID:         "test",
					DefaultPeriodType: "down",
					Resources:         tt.resources,
				},
			}

			hub := &v1alpha3.Gcp{}
			require.NoError(t, src.ConvertTo(hub))

			result := &Gcp{}
			require.NoError(t, result.ConvertFrom(hub))

			assert.Empty(t, result.Spec.Resources.Types)
		})
	}
}

func TestGcpConversion_TypeAssertionError(t *testing.T) {
	src := &Gcp{}
	wrongHub := &v1alpha3.K8s{}

	err := src.ConvertTo(wrongHub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *kubecloudscalercloudv1alpha3.Gcp")

	err = src.ConvertFrom(wrongHub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *kubecloudscalercloudv1alpha3.Gcp")
}
