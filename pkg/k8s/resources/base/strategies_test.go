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

package base

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// mockResourceItem implements ResourceItem for testing.
// Used by both strategies_test.go and processor_test.go.
type mockResourceItem struct {
	name        string
	namespace   string
	annotations map[string]string
}

func (m *mockResourceItem) GetName() string                    { return m.name }
func (m *mockResourceItem) GetNamespace() string               { return m.namespace }
func (m *mockResourceItem) GetAnnotations() map[string]string  { return m.annotations }
func (m *mockResourceItem) SetAnnotations(a map[string]string) { m.annotations = a }

func testLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

func newTestPeriod() *periodPkg.Period {
	return &periodPkg.Period{
		Type:        common.PeriodTypeDown,
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
		Spec:        &common.RecurringPeriod{Timezone: ptr.To("UTC")},
		MinReplicas: 0,
		MaxReplicas: 5,
	}
}

// ---------------------------------------------------------------------------
// IntReplicasStrategy
// ---------------------------------------------------------------------------

func TestIntReplicasStrategy_ApplyScaling(t *testing.T) {
	annotationMgr := utils.NewAnnotationManager()

	tests := []struct {
		name            string
		periodType      string
		period          *periodPkg.Period
		initReplicas    int32
		initAnnotations map[string]string
		wantReplicas    int32
		wantRestored    bool
		wantErr         bool
		wantAnnotation  bool // expect kubecloudscaler annotations present after call
	}{
		{
			name:           "down: saves original and sets minReplicas",
			periodType:     "down",
			period:         newTestPeriod(),
			initReplicas:   3,
			wantReplicas:   0, // period.MinReplicas
			wantRestored:   false,
			wantAnnotation: true,
		},
		{
			name:           "up: saves original and sets maxReplicas",
			periodType:     "up",
			period:         newTestPeriod(),
			initReplicas:   3,
			wantReplicas:   5, // period.MaxReplicas
			wantRestored:   false,
			wantAnnotation: true,
		},
		{
			name:       "restore: reads saved value from annotations and removes annotations",
			periodType: "restore",
			period:     nil,
			initAnnotations: map[string]string{
				"kubecloudscaler.cloud/original-value":    "3",
				"kubecloudscaler.cloud/period-type":       "down",
				"kubecloudscaler.cloud/period-start-time": "2024-01-01T00:00:00Z",
				"kubecloudscaler.cloud/period-end-time":   "2024-01-01T08:00:00Z",
			},
			initReplicas: 0,
			wantReplicas: 3,
			wantRestored: false,
		},
		{
			name:            "restore: already restored (no annotations) returns true",
			periodType:      "restore",
			period:          nil,
			initAnnotations: map[string]string{},
			initReplicas:    3,
			wantReplicas:    3, // unchanged
			wantRestored:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replicas := ptr.To(tt.initReplicas)
			annotations := make(map[string]string)
			for k, v := range tt.initAnnotations {
				annotations[k] = v
			}

			resource := &mockResourceItem{
				name:        "test-deploy",
				namespace:   "default",
				annotations: annotations,
			}

			strategy := NewIntReplicasStrategy(
				"Deployment",
				func(_ ResourceItem) *int32 { return replicas },
				func(_ ResourceItem, v *int32) { replicas = v },
				testLogger(),
				annotationMgr,
			)

			assert.Equal(t, "Deployment", strategy.GetKind())

			restored, err := strategy.ApplyScaling(context.Background(), resource, tt.periodType, tt.period)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRestored, restored)
			assert.Equal(t, tt.wantReplicas, *replicas)

			if tt.wantAnnotation {
				assert.Contains(t, resource.GetAnnotations(), "kubecloudscaler.cloud/original-value")
				assert.Contains(t, resource.GetAnnotations(), "kubecloudscaler.cloud/period-type")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MinMaxReplicasStrategy
// ---------------------------------------------------------------------------

func TestMinMaxReplicasStrategy_ApplyScaling(t *testing.T) {
	annotationMgr := utils.NewAnnotationManager()

	tests := []struct {
		name            string
		periodType      string
		period          *periodPkg.Period
		initMin         int32
		initMax         int32
		initAnnotations map[string]string
		wantMin         int32
		wantMax         int32
		wantRestored    bool
		wantErr         bool
		wantAnnotation  bool
	}{
		{
			name:           "down: saves original min/max and sets period values",
			periodType:     "down",
			period:         newTestPeriod(),
			initMin:        2,
			initMax:        10,
			wantMin:        0, // period.MinReplicas
			wantMax:        5, // period.MaxReplicas
			wantRestored:   false,
			wantAnnotation: true,
		},
		{
			name:           "up: saves original min/max and sets period values",
			periodType:     "up",
			period:         newTestPeriod(),
			initMin:        2,
			initMax:        10,
			wantMin:        0,
			wantMax:        5,
			wantRestored:   false,
			wantAnnotation: true,
		},
		{
			name:       "restore: reads saved values and removes annotations",
			periodType: "restore",
			period:     nil,
			initAnnotations: map[string]string{
				"kubecloudscaler.cloud/min-original-value": "2",
				"kubecloudscaler.cloud/max-original-value": "10",
				"kubecloudscaler.cloud/period-type":        "down",
				"kubecloudscaler.cloud/period-start-time":  "2024-01-01T00:00:00Z",
				"kubecloudscaler.cloud/period-end-time":    "2024-01-01T08:00:00Z",
			},
			initMin:      0,
			initMax:      5,
			wantMin:      2,
			wantMax:      10,
			wantRestored: false,
		},
		{
			name:            "restore: already restored returns true",
			periodType:      "restore",
			period:          nil,
			initAnnotations: map[string]string{},
			initMin:         2,
			initMax:         10,
			wantMin:         2,  // unchanged
			wantMax:         10, // unchanged
			wantRestored:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minReplicas := ptr.To(tt.initMin)
			maxReplicas := ptr.To(tt.initMax)
			annotations := make(map[string]string)
			for k, v := range tt.initAnnotations {
				annotations[k] = v
			}

			resource := &mockResourceItem{
				name:        "test-hpa",
				namespace:   "default",
				annotations: annotations,
			}

			strategy := NewMinMaxReplicasStrategy(
				"HPA",
				func(_ ResourceItem) (*int32, *int32) { return minReplicas, maxReplicas },
				func(_ ResourceItem, min, max *int32) { minReplicas = min; maxReplicas = max },
				testLogger(),
				annotationMgr,
			)

			assert.Equal(t, "HPA", strategy.GetKind())

			restored, err := strategy.ApplyScaling(context.Background(), resource, tt.periodType, tt.period)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRestored, restored)
			assert.Equal(t, tt.wantMin, *minReplicas)
			assert.Equal(t, tt.wantMax, *maxReplicas)

			if tt.wantAnnotation {
				assert.Contains(t, resource.GetAnnotations(), "kubecloudscaler.cloud/min-original-value")
				assert.Contains(t, resource.GetAnnotations(), "kubecloudscaler.cloud/max-original-value")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BoolSuspendStrategy
// ---------------------------------------------------------------------------

func TestBoolSuspendStrategy_ApplyScaling(t *testing.T) {
	annotationMgr := utils.NewAnnotationManager()

	tests := []struct {
		name            string
		periodType      string
		period          *periodPkg.Period
		initSuspend     bool
		suspended       bool // strategy's suspended field
		onUpError       func(ResourceItem) error
		initAnnotations map[string]string
		wantSuspend     bool
		wantRestored    bool
		wantErr         bool
		wantAnnotation  bool
	}{
		{
			name:           "down: saves original and sets suspended",
			periodType:     "down",
			period:         newTestPeriod(),
			initSuspend:    false,
			suspended:      true,
			wantSuspend:    true,
			wantRestored:   false,
			wantAnnotation: true,
		},
		{
			name:           "up: saves original and sets !suspended",
			periodType:     "up",
			period:         newTestPeriod(),
			initSuspend:    true,
			suspended:      true,
			wantSuspend:    false, // !suspended
			wantRestored:   false,
			wantAnnotation: true,
		},
		{
			name:       "up: onUpError returns error",
			periodType: "up",
			period:     newTestPeriod(),
			suspended:  true,
			onUpError: func(r ResourceItem) error {
				return fmt.Errorf("scale up not supported for %s", r.GetName())
			},
			initSuspend: false,
			wantSuspend: false, // unchanged
			wantErr:     true,
		},
		{
			name:       "restore: reads saved value and removes annotations",
			periodType: "restore",
			period:     nil,
			suspended:  true,
			initAnnotations: map[string]string{
				"kubecloudscaler.cloud/original-value":    "false",
				"kubecloudscaler.cloud/period-type":       "down",
				"kubecloudscaler.cloud/period-start-time": "2024-01-01T00:00:00Z",
				"kubecloudscaler.cloud/period-end-time":   "2024-01-01T08:00:00Z",
			},
			initSuspend:  true,
			wantSuspend:  false,
			wantRestored: false,
		},
		{
			name:            "restore: already restored returns true",
			periodType:      "restore",
			period:          nil,
			suspended:       true,
			initAnnotations: map[string]string{},
			initSuspend:     false,
			wantSuspend:     false, // unchanged
			wantRestored:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suspend := ptr.To(tt.initSuspend)
			annotations := make(map[string]string)
			for k, v := range tt.initAnnotations {
				annotations[k] = v
			}

			resource := &mockResourceItem{
				name:        "test-cronjob",
				namespace:   "default",
				annotations: annotations,
			}

			strategy := NewBoolSuspendStrategy(
				"CronJob",
				func(_ ResourceItem) *bool { return suspend },
				func(_ ResourceItem, v *bool) { suspend = v },
				tt.suspended,
				testLogger(),
				tt.onUpError,
				annotationMgr,
			)

			assert.Equal(t, "CronJob", strategy.GetKind())

			restored, err := strategy.ApplyScaling(context.Background(), resource, tt.periodType, tt.period)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRestored, restored)
			assert.Equal(t, tt.wantSuspend, *suspend)

			if tt.wantAnnotation {
				assert.Contains(t, resource.GetAnnotations(), "kubecloudscaler.cloud/original-value")
				assert.Contains(t, resource.GetAnnotations(), "kubecloudscaler.cloud/period-type")
			}
		})
	}
}
