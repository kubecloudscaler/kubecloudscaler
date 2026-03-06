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

// Package metrics provides Prometheus metrics for kubecloudscaler controllers.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "kubecloudscaler"

	LabelController   = "controller"
	LabelResult       = "result"
	LabelResourceType = "resource_type"
	LabelAction       = "action"
	LabelOutcome      = "outcome"

	ResultSuccess          = "success"
	ResultCriticalError    = "critical_error"
	ResultRecoverableError = "recoverable_error"
	ResultError            = "error"
	ResultFailure          = "failure"

	OutcomeActive      = "active"
	OutcomeNoaction    = "noaction"
	OutcomeRunOnceSkip = "run_once_skip"
	OutcomeError       = "error"

	ControllerK8s  = "k8s"
	ControllerGCP  = "gcp"
	ControllerFlow = "flow"
)

var (
	// ReconcileTotal counts reconciliation attempts per controller and result.
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "reconcile_total",
			Help:      "Total number of reconciliations.",
		},
		[]string{LabelController, LabelResult},
	)

	// ReconcileDurationSeconds tracks the duration of full reconciliation cycles.
	ReconcileDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "reconcile_duration_seconds",
			Help:      "Duration of reconciliation in seconds.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{LabelController},
	)

	// ScalingOperationsTotal counts individual resource scaling operations.
	ScalingOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "scaling_operations_total",
			Help:      "Total number of individual resource scaling operations.",
		},
		[]string{LabelController, LabelResourceType, LabelAction, LabelResult},
	)

	// PeriodEvaluationTotal counts period evaluation outcomes.
	PeriodEvaluationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "period_evaluation_total",
			Help:      "Total number of period evaluations.",
		},
		[]string{LabelController, LabelOutcome},
	)

	allCollectors = []prometheus.Collector{
		ReconcileTotal,
		ReconcileDurationSeconds,
		ScalingOperationsTotal,
		PeriodEvaluationTotal,
	}
)

// Register registers all custom metrics with the provided registerer.
// Must be called once during application startup.
func Register(registerer prometheus.Registerer) {
	for _, c := range allCollectors {
		registerer.MustRegister(c)
	}
}
