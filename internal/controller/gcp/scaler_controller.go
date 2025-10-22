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
	"errors"
	"time"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

const (
	// RequeueDelaySeconds is the delay in seconds before requeuing a run-once period.
	RequeueDelaySeconds = 5
)

// ScalerReconciler reconciles a Scaler object
// It manages the lifecycle of GCP resources by scaling them up/down based on configured periods
type ScalerReconciler struct {
	client.Client                 // Kubernetes client for API operations
	Scheme        *runtime.Scheme // Scheme for type conversion and serialization
	Logger        *zerolog.Logger
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
//nolint:gocognit,gocyclo,funlen // Reconcile function complexity is acceptable for controller logic
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Scaler object from the Kubernetes API
	scaler := &kubecloudscalerv1alpha3.Gcp{}
	if err := r.Get(ctx, req.NamespacedName, scaler); err != nil {
		r.Logger.Error().Err(err).Msg("unable to fetch Scaler")

		return ctrl.Result{RequeueAfter: 0 * time.Second}, client.IgnoreNotFound(err)
	}

	r.Logger.Info().Str("name", scaler.Name).Str("kind", scaler.Kind).Str("apiVersion", scaler.APIVersion).Msg("reconciling scaler")

	// Finalizer management for proper cleanup
	scalerFinalizer := "kubecloudscaler.cloud/finalizer"
	scalerFinalize := false

	// Check if the object is being deleted by examining the DeletionTimestamp
	if scaler.DeletionTimestamp.IsZero() {
		// Object is not being deleted - ensure finalizer is present
		// The finalizer ensures we can perform cleanup operations before deletion
		if !controllerutil.ContainsFinalizer(scaler, scalerFinalizer) {
			r.Logger.Info().Msg("adding finalizer")
			controllerutil.AddFinalizer(scaler, scalerFinalizer)
			if err := r.Update(ctx, scaler); err != nil {
				return ctrl.Result{RequeueAfter: 0 * time.Second}, client.IgnoreNotFound(err)
			}
		}
	} else {
		// Object is being deleted - handle finalizer cleanup
		if controllerutil.ContainsFinalizer(scaler, scalerFinalizer) {
			r.Logger.Info().Msg("deleting scaler with finalizer")
			scalerFinalize = true
		} else {
			// Finalizer already removed, stop reconciliation
			return ctrl.Result{RequeueAfter: 0 * time.Second}, nil
		}
	}

	// Handle authentication secret for GCP access
	secret := &corev1.Secret{}
	if scaler.Spec.Config.AuthSecret != nil {
		r.Logger.Info().Msg("auth secret found for GCP authentication")
		// Construct the namespaced name for the secret
		namespacedSecret := types.NamespacedName{
			Namespace: req.Namespace,
			Name:      *scaler.Spec.Config.AuthSecret,
		}

		// Fetch the secret from the cluster
		if err := r.Get(ctx, namespacedSecret, secret); err != nil {
			r.Logger.Error().Err(err).Msg("unable to fetch secret")
		}
	} else {
		// No authentication secret specified, use default GCP access
		secret = nil
	}

	// Initialize GCP client for resource operations
	// This client is used to interact with the GCP Compute Engine API
	//nolint:gocritic // gcpClient variable name intentionally shadows imported package for clarity
	gcpClient, err := gcpClient.GetClient(secret, scaler.Spec.Config.ProjectID)
	if err != nil {
		r.Logger.Error().Err(err).Msg("unable to get GCP client")

		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	// Configure resource management settings
	resourceConfig := resources.Config{
		GCP: &gcpUtils.Config{
			Client:            gcpClient,
			ProjectID:         scaler.Spec.Config.ProjectID,
			Region:            scaler.Spec.Config.Region,
			Names:             scaler.Spec.Resources.Names,
			LabelSelector:     scaler.Spec.Resources.LabelSelector,
			DefaultPeriodType: scaler.Spec.Config.DefaultPeriodType,
		},
	}

	// Convert []common.ScalerPeriod to []*common.ScalerPeriod for utils.ValidatePeriod
	periods := make([]*common.ScalerPeriod, len(scaler.Spec.Periods))
	for i := range scaler.Spec.Periods {
		periods[i] = &scaler.Spec.Periods[i]
	}

	// Validate and determine the current time period for scaling operations
	// This determines whether resources should be scaled up or down based on the current time
	resourceConfig.GCP.Period, err = utils.ValidatePeriod(
		periods,        // Configured time periods
		&scaler.Status, // Current status for tracking
		scaler.Spec.Config.RestoreOnDelete && scalerFinalize, // Restore original state on deletion
	)
	if err != nil {
		// Handle run-once period - requeue until the period ends
		if errors.Is(err, utils.ErrRunOncePeriod) {
			return ctrl.Result{RequeueAfter: time.Until(resourceConfig.GCP.Period.GetEndTime.Add(RequeueDelaySeconds * time.Second))}, nil
		}

		// Update status with error information
		scaler.Status.Comments = ptr.To(err.Error())

		if err := r.Status().Update(ctx, scaler); err != nil {
			r.Logger.Error().Err(err).Msg("unable to update scaler status")
		}

		return ctrl.Result{Requeue: false}, nil
	}

	// Track results of scaling operations
	var (
		recSuccess []common.ScalerStatusSuccess // Successfully scaled resources
		recFailed  []common.ScalerStatusFailed  // Failed scaling operations
	)

	// Validate and filter the list of resources to be scaled
	// This ensures only valid resource types are processed
	resourceList := r.validResourceList(scaler)

	r.Logger.Debug().Msgf("resourceList: %v", resourceList)

	// Process each resource type and perform scaling operations
	for _, resource := range resourceList {
		// Create a resource handler for the specific resource type
		curResource, err := resources.NewResource(resource, resourceConfig, r.Logger)
		if err != nil {
			r.Logger.Error().Err(err).Msg("unable to get resource")

			continue
		}

		// Execute the scaling operation for this resource type
		// This will scale all matching resources up or down based on the current period
		success, failed, err := curResource.SetState(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Msg("unable to set resource state")

			continue
		}

		// Collect results for status reporting
		recSuccess = append(recSuccess, success...)
		recFailed = append(recFailed, failed...)
	}

	// Handle finalizer cleanup if the object is being deleted
	if scalerFinalize {
		r.Logger.Info().Msg("removing finalizer")
		controllerutil.RemoveFinalizer(scaler, scalerFinalizer)
		if err := r.Update(ctx, scaler); err != nil {
			return ctrl.Result{RequeueAfter: 0 * time.Second}, client.IgnoreNotFound(err)
		}

		return ctrl.Result{RequeueAfter: 0 * time.Second}, nil
	}

	// Update the scaler status with operation results
	scaler.Status.CurrentPeriod.Successful = recSuccess
	scaler.Status.CurrentPeriod.Failed = recFailed
	scaler.Status.Comments = ptr.To("time period processed")

	// Persist status updates to the cluster
	if err := r.Status().Update(ctx, scaler); err != nil {
		r.Logger.Error().Err(err).Msg("unable to update scaler status")
	} else {
		r.Logger.Info().Str("name", scaler.Name).Str("kind", scaler.Kind).Str("apiVersion", scaler.APIVersion).Msg("scaler status updated")
	}

	// Requeue for the next reconciliation cycle
	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil
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

// validResourceList validates and filters the list of resources to be scaled.
// It ensures that only valid resource types are included for GCP resources.
func (r *ScalerReconciler) validResourceList(scaler *kubecloudscalerv1alpha3.Gcp) []string {
	// Default to compute instances if no resources are specified
	if len(scaler.Spec.Resources.Types) == 0 {
		scaler.Spec.Resources.Types = []string{resources.DefaultGCPResourceType}
	}

	return scaler.Spec.Resources.Types
}
