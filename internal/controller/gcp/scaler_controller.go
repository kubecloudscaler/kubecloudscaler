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
	Chain         service.Chain // Handler chain for reconciliation
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// This function handles the scaling of GCP resources based on configured time periods.
//
// The reconciliation process:
// 1. Fetches the Scaler object from the cluster
// 2. Manages finalizers for proper cleanup
// 3. Validates and processes authentication secrets
// 4. Determines the current time period and validates it
// 5. Scales resources according to the period configuration
// 6. Updates the status with results
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
//
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

	// Initialize chain if not set (for backward compatibility with tests)
	if r.Chain == nil {
		r.Chain = r.initializeChain()
	}

	// Create reconciliation context
	reconCtx := &service.ReconciliationContext{
		Request: req,
		Client:  r.Client,
		Logger:  r.Logger,
	}

	// Execute handler chain
	result, err := r.Chain.Execute(reconCtx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("handler chain execution failed")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, client.IgnoreNotFound(err)
	}

	return result, nil
}

// initializeChain creates and configures the handler chain for reconciliation.
// This method registers handlers in the fixed order required for GCP scaler reconciliation.
// It uses the Chain of Responsibility pattern by linking handlers together with setNext().
func (r *ScalerReconciler) initializeChain() service.Chain {
	// Create handlers
	statusHandler := handlers.NewStatusHandler()
	scalingHandler := handlers.NewScalingHandler()
	periodHandler := handlers.NewPeriodHandler()
	authHandler := handlers.NewAuthHandler()
	finalizerHandler := handlers.NewFinalizerHandler()
	fetchHandler := handlers.NewFetchHandler()

	// Set next for each handler in reverse order (building the chain backwards)
	scalingHandler.SetNext(statusHandler)
	periodHandler.SetNext(scalingHandler)
	authHandler.SetNext(periodHandler)
	finalizerHandler.SetNext(authHandler)
	fetchHandler.SetNext(finalizerHandler)

	// Return chain starting with fetch handler
	return service.NewHandlerChain([]service.Handler{fetchHandler}, r.Logger)
}

// SetupWithManager sets up the controller with the Manager.
// This method configures the controller to watch for GCP Scaler resources
// and defines the reconciliation behavior.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Gcp{}).              // Watch for GCP Scaler resources
		WithEventFilter(utils.IgnoreDeletionPredicate()). // Filter out deletion events
		Named("gcpScaler").                               // Set controller name
		Complete(r)                                       // Complete the controller setup
}
