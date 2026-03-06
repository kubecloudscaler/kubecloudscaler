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

package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/metrics"
)

func TestReconcileTotal_Increment(t *testing.T) {
	controllers := []string{metrics.ControllerK8s, metrics.ControllerGCP, metrics.ControllerFlow}
	results := []string{metrics.ResultSuccess, metrics.ResultCriticalError, metrics.ResultRecoverableError, metrics.ResultError}

	for _, ctrl := range controllers {
		for _, result := range results {
			before := testutil.ToFloat64(metrics.ReconcileTotal.WithLabelValues(ctrl, result))
			metrics.ReconcileTotal.WithLabelValues(ctrl, result).Inc()
			after := testutil.ToFloat64(metrics.ReconcileTotal.WithLabelValues(ctrl, result))
			assert.Equal(t, before+1, after, "controller=%s result=%s", ctrl, result)
		}
	}
}

func TestReconcileDurationSeconds_Observe(t *testing.T) {
	for _, ctrl := range []string{metrics.ControllerK8s, metrics.ControllerGCP, metrics.ControllerFlow} {
		observer := metrics.ReconcileDurationSeconds.WithLabelValues(ctrl)
		assert.NotNil(t, observer, "controller=%s", ctrl)
		observer.Observe(0.42)
	}
}

func TestScalingOperationsTotal_Increment(t *testing.T) {
	cases := []struct {
		controller   string
		resourceType string
		action       string
		result       string
	}{
		{metrics.ControllerK8s, "deployments", "up", metrics.ResultSuccess},
		{metrics.ControllerK8s, "statefulsets", "down", metrics.ResultFailure},
		{metrics.ControllerK8s, "cronjobs", "up", metrics.ResultSuccess},
		{metrics.ControllerGCP, "vm-instances", "down", metrics.ResultSuccess},
		{metrics.ControllerGCP, "vm-instances", "up", metrics.ResultFailure},
	}

	for _, tc := range cases {
		before := testutil.ToFloat64(
			metrics.ScalingOperationsTotal.WithLabelValues(tc.controller, tc.resourceType, tc.action, tc.result),
		)
		metrics.ScalingOperationsTotal.WithLabelValues(tc.controller, tc.resourceType, tc.action, tc.result).Inc()
		after := testutil.ToFloat64(
			metrics.ScalingOperationsTotal.WithLabelValues(tc.controller, tc.resourceType, tc.action, tc.result),
		)
		assert.Equal(t, before+1, after, "%+v", tc)
	}
}

func TestPeriodEvaluationTotal_Increment(t *testing.T) {
	controllers := []string{metrics.ControllerK8s, metrics.ControllerGCP}
	outcomes := []string{metrics.OutcomeActive, metrics.OutcomeNoaction, metrics.OutcomeRunOnceSkip, metrics.OutcomeError}

	for _, ctrl := range controllers {
		for _, outcome := range outcomes {
			before := testutil.ToFloat64(metrics.PeriodEvaluationTotal.WithLabelValues(ctrl, outcome))
			metrics.PeriodEvaluationTotal.WithLabelValues(ctrl, outcome).Inc()
			after := testutil.ToFloat64(metrics.PeriodEvaluationTotal.WithLabelValues(ctrl, outcome))
			assert.Equal(t, before+1, after, "controller=%s outcome=%s", ctrl, outcome)
		}
	}
}
