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

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// FlowReconciler reconciles a Flow object
// It manages the lifecycle of K8s and GCP resources by creating and deploying them
// based on configured periods and flow definitions with timing delays
type FlowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger *zerolog.Logger
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=flows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=flows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=flows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// This function handles the creation and deployment of K8s and GCP resources
// based on flow definitions with timing delays.
//
// The reconciliation process:
// 1. Fetches the Flow object from the cluster
// 2. Manages finalizers for proper cleanup
// 3. Validates flow definitions and timing constraints
// 4. Creates K8s and GCP objects with owner references
// 5. Deploys the created objects
// 6. Updates the status with results
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *FlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Fetch the Flow object from the Kubernetes API
	flow := &kubecloudscalerv1alpha3.Flow{}
	if err := r.Get(ctx, req.NamespacedName, flow); err != nil {
		r.Logger.Error().Err(err).Msg("unable to fetch Flow")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Logger.Info().
		Str("name", flow.Name).
		Str("kind", flow.Kind).
		Str("apiVersion", flow.APIVersion).
		Msg("reconciling flow")

	// Finalizer management for proper cleanup
	flowFinalizer := "kubecloudscaler.cloud/flow-finalizer"
	flowFinalize := false

	// Check if the object is being deleted
	if flow.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object is not being deleted - ensure finalizer is present
		if !controllerutil.ContainsFinalizer(flow, flowFinalizer) {
			r.Logger.Info().Msg("adding finalizer")
			controllerutil.AddFinalizer(flow, flowFinalizer)
			if err := r.Update(ctx, flow); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
	} else {
		// Object is being deleted - handle finalizer cleanup
		if controllerutil.ContainsFinalizer(flow, flowFinalizer) {
			r.Logger.Info().Msg("deleting flow with finalizer")
			flowFinalize = true
		} else {
			// Finalizer already removed, stop reconciliation
			return ctrl.Result{}, nil
		}
	}

	// Handle finalizer cleanup if the object is being deleted
	if flowFinalize {
		r.Logger.Info().Msg("removing finalizer")
		controllerutil.RemoveFinalizer(flow, flowFinalizer)
		if err := r.Update(ctx, flow); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, nil
	}

	// Flow validation is now handled by the webhook

	// Process flow and create resources
	if err := r.processFlow(ctx, flow); err != nil {
		// Check if this is a scheduling error (requeue needed)
		if strings.Contains(err.Error(), "scheduled for") && strings.Contains(err.Error(), "requeue after") {
			// Extract requeue duration from error message
			parts := strings.Split(err.Error(), "requeue after ")
			if len(parts) > 1 {
				requeueStr := strings.TrimSpace(parts[1])
				if requeueDuration, parseErr := time.ParseDuration(requeueStr); parseErr == nil {
					r.Logger.Info().
						Str("flow", flow.Name).
						Dur("requeueAfter", requeueDuration).
						Msg("flow scheduled for later processing")
					return ctrl.Result{RequeueAfter: requeueDuration}, nil
				}
			}
		}

		r.Logger.Error().Err(err).Msg("flow processing failed")
		return r.updateFlowStatus(ctx, flow, metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionFalse,
			Reason:  "ProcessingFailed",
			Message: err.Error(),
		})
	}

	// Update status to indicate successful processing
	return r.updateFlowStatus(ctx, flow, metav1.Condition{
		Type:    "Processed",
		Status:  metav1.ConditionTrue,
		Reason:  "ProcessingSucceeded",
		Message: "Flow processed successfully",
	})
}

// processFlow processes the flow definition and creates/deploys resources
func (r *FlowReconciler) processFlow(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow) error {
	// Convert periods to map for easier lookup
	periodsMap := make(map[string]*common.ScalerPeriod)
	for _, period := range flow.Spec.Periods {
		periodName := ptr.Deref(period.Name, "")
		periodsMap[periodName] = period
	}

	// Convert K8s resources to map for easier lookup
	k8sResourcesMap := make(map[string]kubecloudscalerv1alpha3.K8sResource)
	for _, resource := range flow.Spec.Resources.K8s {
		k8sResourcesMap[resource.Name] = resource
	}

	// Convert GCP resources to map for easier lookup
	gcpResourcesMap := make(map[string]kubecloudscalerv1alpha3.GcpResource)
	for _, resource := range flow.Spec.Resources.Gcp {
		gcpResourcesMap[resource.Name] = resource
	}

	// Process K8s resources with timing delays
	for _, k8sResource := range flow.Spec.Resources.K8s {
		if err := r.createAndDeployK8sResourceWithTiming(ctx, flow, k8sResource, periodsMap); err != nil {
			return fmt.Errorf("failed to create K8s resource %s: %w", k8sResource.Name, err)
		}
	}

	// Process GCP resources with timing delays
	for _, gcpResource := range flow.Spec.Resources.Gcp {
		if err := r.createAndDeployGcpResourceWithTiming(ctx, flow, gcpResource, periodsMap); err != nil {
			return fmt.Errorf("failed to create GCP resource %s: %w", gcpResource.Name, err)
		}
	}

	return nil
}

// createAndDeployK8sResourceWithTiming creates and deploys a K8s resource with timing delays
func (r *FlowReconciler) createAndDeployK8sResourceWithTiming(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, k8sResource kubecloudscalerv1alpha3.K8sResource, periodsMap map[string]*common.ScalerPeriod) error {
	// Find the flow definition for this resource
	var resourceFlow *kubecloudscalerv1alpha3.Flows
	for _, f := range flow.Spec.Flows {
		for _, resource := range f.Resources {
			if resource.Name == k8sResource.Name {
				resourceFlow = &f
				break
			}
		}
		if resourceFlow != nil {
			break
		}
	}

	// If no flow definition found, create immediately
	if resourceFlow == nil {
		r.Logger.Debug().
			Str("resource", k8sResource.Name).
			Msg("no flow definition found for resource, creating immediately")
		return nil
	}

	// Find the period for this flow using the map
	targetPeriod, exists := periodsMap[resourceFlow.PeriodName]
	if !exists {
		return fmt.Errorf("period %s not found for resource %s", resourceFlow.PeriodName, k8sResource.Name)
	}

	// Calculate the delay for this specific resource
	var resourceDelay time.Duration
	for _, resource := range resourceFlow.Resources {
		if resource.Name == k8sResource.Name {
			if resource.Delay != nil {
				delay, err := time.ParseDuration(*resource.Delay)
				if err != nil {
					return fmt.Errorf("invalid delay format for resource %s: %w", resource.Name, err)
				}
				resourceDelay = delay
			}
			break
		}
	}

	// Calculate the scheduled start time (period start + delay)
	scheduledStartTime, err := r.calculateScheduledStartTime(targetPeriod, resourceDelay)
	if err != nil {
		return fmt.Errorf("failed to calculate scheduled start time: %w", err)
	}

	// Check if it's time to deploy this resource
	if time.Now().Before(scheduledStartTime) {
		// Schedule for later deployment
		requeueAfter := time.Until(scheduledStartTime)
		r.Logger.Info().
			Str("resource", k8sResource.Name).
			Time("scheduledTime", scheduledStartTime).
			Dur("requeueAfter", requeueAfter).
			Msg("resource scheduled for later deployment")

		// Return a special error to indicate requeue is needed
		return fmt.Errorf("resource %s scheduled for %v, requeue after %v", k8sResource.Name, scheduledStartTime, requeueAfter)
	}

	// It's time to deploy, create the resource
	return r.createAndDeployK8sResource(ctx, flow, k8sResource)
}

// createAndDeployGcpResourceWithTiming creates and deploys a GCP resource with timing delays
func (r *FlowReconciler) createAndDeployGcpResourceWithTiming(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, gcpResource kubecloudscalerv1alpha3.GcpResource, periodsMap map[string]*common.ScalerPeriod) error {
	// Find the flow definition for this resource
	var resourceFlow *kubecloudscalerv1alpha3.Flows
	for _, f := range flow.Spec.Flows {
		for _, resource := range f.Resources {
			if resource.Name == gcpResource.Name {
				resourceFlow = &f
				break
			}
		}
		if resourceFlow != nil {
			break
		}
	}

	// If no flow definition found, create immediately
	if resourceFlow == nil {
		r.Logger.Debug().
			Str("resource", gcpResource.Name).
			Msg("no flow definition found for resource, creating immediately")
		return nil
	}

	// Find the period for this flow using the map
	targetPeriod, exists := periodsMap[resourceFlow.PeriodName]
	if !exists {
		return fmt.Errorf("period %s not found for resource %s", resourceFlow.PeriodName, gcpResource.Name)
	}

	// Calculate the delay for this specific resource
	var resourceDelay time.Duration
	for _, resource := range resourceFlow.Resources {
		if resource.Name == gcpResource.Name {
			if resource.Delay != nil {
				delay, err := time.ParseDuration(*resource.Delay)
				if err != nil {
					return fmt.Errorf("invalid delay format for resource %s: %w", resource.Name, err)
				}
				resourceDelay = delay
			}
			break
		}
	}

	// Calculate the scheduled start time (period start + delay)
	scheduledStartTime, err := r.calculateScheduledStartTime(targetPeriod, resourceDelay)
	if err != nil {
		return fmt.Errorf("failed to calculate scheduled start time: %w", err)
	}

	// Check if it's time to deploy this resource
	if time.Now().Before(scheduledStartTime) {
		// Schedule for later deployment
		requeueAfter := time.Until(scheduledStartTime)
		r.Logger.Info().
			Str("resource", gcpResource.Name).
			Time("scheduledTime", scheduledStartTime).
			Dur("requeueAfter", requeueAfter).
			Msg("resource scheduled for later deployment")

		// Return a special error to indicate requeue is needed
		return fmt.Errorf("resource %s scheduled for %v, requeue after %v", gcpResource.Name, scheduledStartTime, requeueAfter)
	}

	// It's time to deploy, create the resource
	return r.createAndDeployGcpResource(ctx, flow, gcpResource)
}

// calculateScheduledStartTime calculates when a resource should be deployed based on period start time and delay
func (r *FlowReconciler) calculateScheduledStartTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error) {
	var periodStartTime time.Time
	var err error

	if period.Time.Recurring != nil {
		// For recurring periods, parse the start time
		periodStartTime, err = time.Parse("15:04", period.Time.Recurring.StartTime)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to get next recurring period start: %w", err)
		}
	} else if period.Time.Fixed != nil {
		// For fixed periods, parse the start time
		periodStartTime, err = time.Parse("2006-01-02 15:04:05", period.Time.Fixed.StartTime)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse fixed period start time: %w", err)
		}
	} else {
		return time.Time{}, errors.New("no valid time period found")
	}

	// Add the delay to the period start time
	return periodStartTime.Add(delay), nil
}

// createAndDeployK8sResource creates and deploys a K8s resource with owner reference
func (r *FlowReconciler) createAndDeployK8sResource(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, k8sResource kubecloudscalerv1alpha3.K8sResource) error {
	// Create K8s object
	k8sObj := &kubecloudscalerv1alpha3.K8s{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", flow.Name, k8sResource.Name),
			Namespace: flow.Namespace,
		},
		Spec: kubecloudscalerv1alpha3.K8sSpec{
			DryRun:    false,
			Periods:   flow.Spec.Periods,
			Resources: k8sResource.Resources,
			Config:    k8sResource.Config,
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(flow, k8sObj, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update the K8s object
	if err := r.Create(ctx, k8sObj); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			return fmt.Errorf("failed to create K8s object: %w", err)
		}
		// Object already exists, update it
		if err := r.Update(ctx, k8sObj); err != nil {
			return fmt.Errorf("failed to update K8s object: %w", err)
		}
	}

	r.Logger.Info().
		Str("name", k8sObj.Name).
		Str("namespace", k8sObj.Namespace).
		Msg("created/updated K8s resource")

	return nil
}

// createAndDeployGcpResource creates and deploys a GCP resource with owner reference
func (r *FlowReconciler) createAndDeployGcpResource(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, gcpResource kubecloudscalerv1alpha3.GcpResource) error {
	// Create GCP object
	gcpObj := &kubecloudscalerv1alpha3.Gcp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", flow.Name, gcpResource.Name),
			Namespace: flow.Namespace,
		},
		Spec: kubecloudscalerv1alpha3.GcpSpec{
			DryRun:    false,
			Periods:   flow.Spec.Periods,
			Resources: gcpResource.Resources,
			Config:    gcpResource.Config,
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(flow, gcpObj, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update the GCP object
	if err := r.Create(ctx, gcpObj); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			return fmt.Errorf("failed to create GCP object: %w", err)
		}
		// Object already exists, update it
		if err := r.Update(ctx, gcpObj); err != nil {
			return fmt.Errorf("failed to update GCP object: %w", err)
		}
	}

	r.Logger.Info().
		Str("name", gcpObj.Name).
		Str("namespace", gcpObj.Namespace).
		Msg("created/updated GCP resource")

	return nil
}

// updateFlowStatus updates the flow status with the given condition
func (r *FlowReconciler) updateFlowStatus(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, condition metav1.Condition) (ctrl.Result, error) {
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	condition.ObservedGeneration = flow.Generation

	// Update or add the condition
	conditionIndex := -1
	for i, c := range flow.Status.Conditions {
		if c.Type == condition.Type {
			conditionIndex = i
			break
		}
	}

	if conditionIndex >= 0 {
		flow.Status.Conditions[conditionIndex] = condition
	} else {
		flow.Status.Conditions = append(flow.Status.Conditions, condition)
	}

	// Update the status
	if err := r.Status().Update(ctx, flow); err != nil {
		r.Logger.Error().Err(err).Msg("unable to update flow status")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, err
	}

	r.Logger.Info().
		Str("name", flow.Name).
		Str("condition", condition.Type).
		Str("status", string(condition.Status)).
		Msg("flow status updated")

	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Flow{}).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Named("flow").
		Complete(r)
}
