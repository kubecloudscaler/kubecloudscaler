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
	scalerService "github.com/kubecloudscaler/kubecloudscaler/internal/controller/scaler/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// ScalerReconciler reconciles a Scaler object
// It manages the lifecycle of GCP resources by scaling them up/down based on configured periods
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger *zerolog.Logger

	// Services for clean architecture
	scalerProcessor *service.ScalerProcessorService
}

// NewScalerReconciler creates a new ScalerReconciler with all required services
func NewScalerReconciler(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	logger *zerolog.Logger,
) *ScalerReconciler {
	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

	// Create services
	periodValidator := scalerService.NewPeriodValidatorService()
	resourceProcessor := scalerService.NewResourceProcessorService(logger)
	scalerProcessor := service.NewScalerProcessorService(periodValidator, resourceProcessor, logger)

	return &ScalerReconciler{
		Client:          k8sClient,
		Scheme:          scheme,
		Logger:          logger,
		scalerProcessor: scalerProcessor,
	}
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
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	scaler := &kubecloudscalerv1alpha3.Gcp{}
	if err := r.Get(ctx, req.NamespacedName, scaler); err != nil {
		r.Logger.Error().Err(err).Msg("unable to fetch Scaler")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Logger.Info().
		Str("name", scaler.Name).
		Str("kind", scaler.Kind).
		Str("apiVersion", scaler.APIVersion).
		Msg("reconciling scaler")

	// Handle finalizer management
	if shouldStop, err := r.handleFinalizers(ctx, scaler); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else if shouldStop {
		return ctrl.Result{}, nil
	}

	// Handle authentication secret
	secret, err := r.handleSecret(ctx, req, scaler)
	if err != nil {
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	// Process scaler
	result, recSuccess, recFailed, err := r.scalerProcessor.ProcessScaler(ctx, scaler, secret, r.isDeleting(scaler))
	if err != nil {
		return r.handleProcessingError(ctx, scaler, err)
	}

	// Update status with results
	if err := utils.UpdateScalerStatus(ctx, r.Status(), scaler, recSuccess, recFailed, "time period processed", r.Logger); err != nil {
		r.Logger.Error().Err(err).Msg("unable to update scaler status")
	}

	return result, nil
}

// handleFinalizers handles finalizer management for proper cleanup.
// Returns true if reconciliation should stop, false otherwise.
func (r *ScalerReconciler) handleFinalizers(ctx context.Context, scaler *kubecloudscalerv1alpha3.Gcp) (bool, error) {
	result := utils.HandleFinalizerReconcile(ctx, r.Client, scaler, utils.DefaultFinalizerName(), r.Logger)
	if result.Error != nil {
		return true, result.Error
	}
	if result.ShouldStop {
		return true, nil
	}
	return false, nil
}

// handleSecret handles authentication secret fetching.
func (r *ScalerReconciler) handleSecret(ctx context.Context, req ctrl.Request, scaler *kubecloudscalerv1alpha3.Gcp) (interface{}, error) {
	secret, err := utils.FetchSecret(ctx, r.Client, scaler.Spec.Config.AuthSecret, req.Namespace, r.Logger)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// isDeleting checks if the scaler is being deleted.
func (r *ScalerReconciler) isDeleting(scaler *kubecloudscalerv1alpha3.Gcp) bool {
	return !scaler.DeletionTimestamp.IsZero()
}

// handleProcessingError handles processing errors and updates status.
func (r *ScalerReconciler) handleProcessingError(ctx context.Context, scaler *kubecloudscalerv1alpha3.Gcp, err error) (ctrl.Result, error) {
	r.Logger.Error().Err(err).Str("scaler", scaler.Name).Msg("scaler processing failed")
	if updateErr := utils.UpdateScalerStatusWithError(ctx, r.Status(), scaler, err, r.Logger); updateErr != nil {
		r.Logger.Error().Err(updateErr).Msg("unable to update scaler status")
	}
	return ctrl.Result{Requeue: false}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Gcp{}).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Named("gcpScaler").
		Complete(r)
}
