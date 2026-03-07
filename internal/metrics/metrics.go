/*
Copyright 2025.

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

// Package metrics provides Prometheus metrics for the kubecloudscaler operator.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamespace = "kubecloudscaler"

	// Controller label values.
	ControllerK8sScaler = "k8s_scaler"
	ControllerGcpScaler = "gcp_scaler"
	ControllerFlow      = "flow"

	// Result label values for reconciliation.
	ResultSuccess          = "success"
	ResultCriticalError    = "critical_error"
	ResultRecoverableError = "recoverable_error"

	// Result label values for scaling operations.
	ScalingSuccess = "success"
	ScalingFailed  = "failed"
)

var (
	initOnce sync.Once
	// DefaultRecorder is the global recorder used by controllers when none is injected.
	DefaultRecorder Recorder = &noopRecorder{}
)

// Recorder records operator metrics. Controllers use this interface for testability.
type Recorder interface {
	RecordReconcile(controller, result string, durationSeconds float64)
	RecordScaling(controller, resourceKind, result string, count int)
	RecordPeriodActive(controller, periodType string)
}

// Init registers custom metrics with the default Prometheus registry.
// Safe to call multiple times; registration runs once.
// Controllers should call Init from main before starting the manager.
func Init() {
	initOnce.Do(registerMetrics)
}

func registerMetrics() {
	registry := prometheus.DefaultRegisterer
	registry.MustRegister(
		reconcileTotal,
		reconcileDurationSeconds,
		scalingOperationsTotal,
		periodActivationsTotal,
	)
	DefaultRecorder = newPromRecorder()
}

var (
	reconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "reconcile_total",
			Help:      "Total number of reconciliations by controller and result (success, critical_error, recoverable_error).",
		},
		[]string{"controller", "result"},
	)

	reconcileDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricNamespace,
			Name:      "reconcile_duration_seconds",
			Help:      "Duration of reconciliation in seconds by controller.",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2.5, 12),
		},
		[]string{"controller"},
	)

	scalingOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "scaling_operations_total",
			Help:      "Total number of scaling operations by controller, resource kind, and result (success, failed).",
		},
		[]string{"controller", "resource_kind", "result"},
	)

	periodActivationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "period_activations_total",
			Help:      "Total number of times a period was set as active by controller and period type (up, down, noaction).",
		},
		[]string{"controller", "period_type"},
	)
)

type promRecorder struct{}

func newPromRecorder() *promRecorder {
	return &promRecorder{}
}

func (p *promRecorder) RecordReconcile(controller, result string, durationSeconds float64) {
	reconcileTotal.WithLabelValues(controller, result).Inc()
	reconcileDurationSeconds.WithLabelValues(controller).Observe(durationSeconds)
}

func (p *promRecorder) RecordScaling(controller, resourceKind, result string, count int) {
	if count <= 0 {
		return
	}
	scalingOperationsTotal.WithLabelValues(controller, resourceKind, result).Add(float64(count))
}

func (p *promRecorder) RecordPeriodActive(controller, periodType string) {
	periodActivationsTotal.WithLabelValues(controller, periodType).Inc()
}

// noopRecorder is used when metrics are disabled or in tests.
type noopRecorder struct{}

func (noopRecorder) RecordReconcile(_, _ string, _ float64) {}

func (noopRecorder) RecordScaling(_, _, _ string, _ int) {}

func (noopRecorder) RecordPeriodActive(_, _ string) {}

// GetRecorder returns the default recorder. After Init(), it returns the Prometheus recorder.
func GetRecorder() Recorder {
	return DefaultRecorder
}

// NormalizePeriodType returns a label-safe period type (up, down, noaction). Empty or unknown becomes "noaction".
func NormalizePeriodType(periodType string) string {
	switch periodType {
	case "up", "down", "noaction":
		return periodType
	default:
		return "noaction"
	}
}

// ScalingResult holds Kind for aggregation (avoids metrics depending on api/common).
type ScalingResult struct {
	Kind string
}

// RecordScalingFromResults aggregates success and failed results by kind and records scaling metrics.
func RecordScalingFromResults(rec Recorder, controller string, success, failed []ScalingResult) {
	successByKind := make(map[string]int)
	for _, s := range success {
		successByKind[s.Kind]++
	}
	for kind, n := range successByKind {
		rec.RecordScaling(controller, kind, ScalingSuccess, n)
	}
	failedByKind := make(map[string]int)
	for _, f := range failed {
		failedByKind[f.Kind]++
	}
	for kind, n := range failedByKind {
		rec.RecordScaling(controller, kind, ScalingFailed, n)
	}
}
