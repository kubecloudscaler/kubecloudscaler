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

// Package gcp provides GCP controller functionality for the kubecloudscaler project.
package gcp

import (
	"context"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// ScalerReconciler reconciles a Scaler object
// It manages the lifecycle of GCP resources by scaling them up/down based on configured periods
type ScalerReconciler struct {
	client.Client                 // Kubernetes client for API operations
	Scheme        *runtime.Scheme // Scheme for type conversion and serialization
	Logger        *zerolog.Logger
	chain         service.Handler // handler chain, initialized once
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// This function delegates to the handler chain for reconciliation logic.
//
// The reconciliation is handled by a Chain of Responsibility pattern with the following handlers:
//  1. Fetch - Fetch scaler resource from Kubernetes API
//  2. Finalizer - Manage finalizer lifecycle
//  3. Authentication - Setup GCP client with authentication
//  4. Period Validation - Validate and determine current time period
//  5. Resource Scaling - Scale GCP resources based on period
//  6. Status Update - Update scaler status with operation results
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger.Info().
		Str("name", req.Name).
		Str("namespace", req.Namespace).
		Msg("reconciling scaler")

	// Create reconciliation context
	reconCtx := &service.ReconciliationContext{
		Ctx:     ctx,
		Request: req,
		Client:  r.Client,
		Logger:  r.Logger,
	}

	// Initialize chain lazily if not set (e.g., in tests without SetupWithManager)
	if r.chain == nil {
		r.chain = r.initializeChain()
	}

	// Execute the handler chain
	err := r.chain.Execute(reconCtx)

	// Handle chain execution result with proper error classification
	if err != nil {
		if service.IsCriticalError(err) {
			r.Logger.Error().Err(err).Msg("critical error during reconciliation")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if service.IsRecoverableError(err) {
			r.Logger.Warn().Err(err).Msg("recoverable error during reconciliation, will requeue")
			requeue := utils.ReconcileErrorDuration
			if reconCtx.RequeueAfter > 0 {
				requeue = reconCtx.RequeueAfter
			}
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		// Unknown error type
		r.Logger.Error().Err(err).Msg("unexpected error during reconciliation")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	// Successful reconciliation - use requeue from context or default
	if reconCtx.RequeueAfter > 0 {
		return ctrl.Result{RequeueAfter: reconCtx.RequeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

// initializeChain creates and links the handler chain in fixed order.
// This follows the classic Chain of Responsibility pattern where handlers
// are linked via SetNext() calls.
//
// Handler Order:
// 1. FetchHandler → 2. FinalizerHandler → 3. AuthHandler →
// 4. PeriodHandler → 5. ScalingHandler → 6. StatusHandler
func (r *ScalerReconciler) initializeChain() service.Handler {
	// Create all handlers
	fetchHandler := handlers.NewFetchHandler()
	finalizerHandler := handlers.NewFinalizerHandler()
	authHandler := handlers.NewAuthHandler()
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
// This method configures the controller to watch for GCP Scaler resources
// and defines the reconciliation behavior.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.chain = r.initializeChain()

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Gcp{}).              // Watch for GCP Scaler resources
		WithEventFilter(utils.IgnoreDeletionPredicate()). // Filter out deletion events
		Named("gcpScaler").                               // Set controller name
		Complete(r)                                       // Complete the controller setup
}
