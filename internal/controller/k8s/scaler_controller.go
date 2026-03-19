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

// Package k8s provides Kubernetes controller functionality for the kubecloudscaler project.
package k8s

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/metrics"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// ScalerReconciler reconciles a Scaler object
// It manages the lifecycle of K8s resources by scaling them up/down based on configured periods.
//
// This controller uses the Chain of Responsibility pattern following the classic
// refactoring.guru Go pattern where handlers have Execute() and SetNext() methods.
// See: https://refactoring.guru/design-patterns/chain-of-responsibility/go/example
type ScalerReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Logger    *zerolog.Logger
	recorder  metrics.Recorder
	chain     service.Handler
	chainOnce sync.Once
}

// NewScalerReconciler creates a new K8s ScalerReconciler with proper dependency injection.
func NewScalerReconciler(c client.Client, scheme *runtime.Scheme, logger *zerolog.Logger, rec metrics.Recorder) *ScalerReconciler {
	if rec == nil {
		rec = metrics.GetRecorder()
	}
	return &ScalerReconciler{
		Client:   c,
		Scheme:   scheme,
		Logger:   logger,
		recorder: rec,
	}
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=k8s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=k8s/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=k8s/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// This function handles the scaling of Kubernetes resources based on configured time periods.
//
// The reconciliation process is implemented as a Chain of Responsibility:
// 1. FetchHandler: Fetches the Scaler object from the cluster
// 2. FinalizerHandler: Manages finalizers for proper cleanup
// 3. AuthHandler: Validates and processes authentication secrets
// 4. PeriodHandler: Determines the current time period and validates it
// 5. ScalingHandler: Scales resources according to the period configuration
// 6. StatusHandler: Updates the status with results
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	rec := r.recorder
	if rec == nil {
		rec = metrics.GetRecorder()
	}

	logger := r.Logger.With().Str("controller", "k8s").Str("name", req.Name).Logger()
	logger.Info().Msg("reconciling scaler")

	// Create reconciliation context with initial values
	reconCtx := &service.ReconciliationContext{
		Ctx:     ctx,
		Request: req,
		Client:  r.Client,
		Logger:  &logger,
	}

	// Initialize chain lazily if not set (e.g., in tests without SetupWithManager)
	r.chainOnce.Do(func() {
		if r.chain == nil {
			r.chain = r.initializeChain()
		}
	})

	// Execute the handler chain
	err := r.chain.Execute(reconCtx)
	duration := time.Since(start).Seconds()

	// Handle chain execution result
	if err != nil {
		if service.IsCriticalError(err) {
			rec.RecordReconcile(metrics.ControllerK8sScaler, metrics.ResultCriticalError, duration)
			logger.Error().Err(err).Msg("critical error during reconciliation")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if service.IsRecoverableError(err) {
			rec.RecordReconcile(metrics.ControllerK8sScaler, metrics.ResultRecoverableError, duration)
			logger.Warn().Err(err).Msg("recoverable error during reconciliation, will requeue")
			requeue := utils.ReconcileErrorDuration
			if reconCtx.RequeueAfter > 0 {
				requeue = reconCtx.RequeueAfter
			}
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		rec.RecordReconcile(metrics.ControllerK8sScaler, metrics.ResultUnclassifiedError, duration)
		logger.Error().Err(err).Msg("unexpected error during reconciliation")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	// Success: record period and scaling metrics, then reconcile result
	if reconCtx.Period != nil {
		rec.RecordPeriodActive(metrics.ControllerK8sScaler,
			metrics.NormalizePeriodType(string(reconCtx.Period.Type)))
	}
	metrics.RecordScalingFromResults(rec, metrics.ControllerK8sScaler,
		toScalingResults(reconCtx.SuccessResults), toScalingResultsFailed(reconCtx.FailedResults))
	rec.RecordReconcile(metrics.ControllerK8sScaler, metrics.ResultSuccess, duration)

	// Successful reconciliation - use requeue from context or default
	if reconCtx.RequeueAfter > 0 {
		return ctrl.Result{RequeueAfter: reconCtx.RequeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func toScalingResults(s []common.ScalerStatusSuccess) []metrics.ScalingResult {
	out := make([]metrics.ScalingResult, 0, len(s))
	for _, r := range s {
		out = append(out, metrics.ScalingResult{Kind: r.Kind})
	}
	return out
}

func toScalingResultsFailed(s []common.ScalerStatusFailed) []metrics.ScalingResult {
	out := make([]metrics.ScalingResult, 0, len(s))
	for _, r := range s {
		out = append(out, metrics.ScalingResult{Kind: r.Kind})
	}
	return out
}

// initializeChain creates and links the handler chain in fixed order.
// This follows the classic Chain of Responsibility pattern from refactoring.guru
// where handlers are linked via SetNext() calls.
//
// Handler Order:
// 1. FetchHandler → 2. FinalizerHandler → 3. AuthHandler →
// 4. PeriodHandler → 5. ScalingHandler → 6. StatusHandler
func (r *ScalerReconciler) initializeChain() service.Handler {
	// Create all handlers
	fetchHandler := handlers.NewFetchHandler()
	finalizerHandler := handlers.NewFinalizerHandler()
	authHandler := handlers.NewAuthHandler(nil)
	periodHandler := handlers.NewPeriodHandler()
	scalingHandler := handlers.NewScalingHandler()
	statusHandler := handlers.NewStatusHandler()

	// Link handlers via SetNext() in fixed order
	fetchHandler.SetNext(finalizerHandler)
	finalizerHandler.SetNext(authHandler)
	authHandler.SetNext(periodHandler)
	periodHandler.SetNext(scalingHandler)
	scalingHandler.SetNext(statusHandler)
	// statusHandler.SetNext(nil) - implicit, last handler

	// Return the first handler in the chain
	return fetchHandler
}

// SetupWithManager sets up the controller with the Manager.
// This method configures the controller to watch for K8s Scaler resources
// and defines the reconciliation behavior.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.chain = r.initializeChain()

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.K8s{}).              // Watch for K8s Scaler resources
		WithEventFilter(utils.IgnoreDeletionPredicate()). // Filter out deletion events
		Named("k8sScaler").                               // Set controller name
		Complete(r)                                       // Complete the controller setup
}
