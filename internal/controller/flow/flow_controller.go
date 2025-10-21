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

// Package controller provides flow controller functionality for the kubecloudscaler project.
package controller

import (
	"context"
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

// ResourceInfo contains information about a resource and its associated periods
type ResourceInfo struct {
	Type     string            // "k8s" or "gcp"
	Resource interface{}       // K8sResource or GcpResource
	Periods  []PeriodWithDelay // Associated periods with delays
}

// PeriodWithDelay contains period information with calculated delay
type PeriodWithDelay struct {
	Period    common.ScalerPeriod
	Delay     time.Duration
	StartTime time.Time
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
//
//nolint:gocognit,gocyclo // Reconcile function complexity is acceptable for controller logic
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
	if flow.DeletionTimestamp.IsZero() {
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
	// Extract all resource names and period names from flows
	resourceNames, periodNames, err := r.extractFlowData(flow)
	if err != nil {
		return fmt.Errorf("failed to extract flow data: %w", err)
	}

	// Validate timing constraints for each period
	if err := r.validatePeriodTimings(flow, periodNames); err != nil {
		return fmt.Errorf("period timing validation failed: %w", err)
	}

	// Create resource mappings for easier lookup
	resourceMappings, err := r.createResourceMappings(flow, resourceNames)
	if err != nil {
		return fmt.Errorf("failed to create resource mappings: %w", err)
	}

	// Process each unique resource
	for resourceName, resourceInfo := range resourceMappings {
		if err := r.processResource(ctx, flow, resourceName, resourceInfo); err != nil {
			return fmt.Errorf("failed to process resource %s: %w", resourceName, err)
		}
	}

	return nil
}

// extractFlowData extracts all resource names and period names from flows
//
//nolint:gocritic,unparam // Multiple return values needed for clear separation of concerns, error for future compatibility
func (r *FlowReconciler) extractFlowData(flow *kubecloudscalerv1alpha3.Flow) (map[string]bool, map[string]bool, error) {
	resourceNames := make(map[string]bool)
	periodNames := make(map[string]bool)

	for _, flowItem := range flow.Spec.Flows {
		// Extract period name
		periodNames[flowItem.PeriodName] = true

		// Extract resource names
		for _, resource := range flowItem.Resources {
			resourceNames[resource.Name] = true
		}
	}

	return resourceNames, periodNames, nil
}

// validatePeriodTimings validates that the sum of delays for each period doesn't exceed the period duration
//
//nolint:gocognit // Validation function complexity is acceptable for comprehensive checks
func (r *FlowReconciler) validatePeriodTimings(flow *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool) error {
	// Create period map for lookup
	periodsMap := make(map[string]common.ScalerPeriod)
	for i := range flow.Spec.Periods {
		period := flow.Spec.Periods[i]
		periodName := ptr.Deref(period.Name, "")
		periodsMap[periodName] = period
	}

	for periodName := range periodNames {
		period, exists := periodsMap[periodName]
		if !exists {
			return fmt.Errorf("period %s referenced in flows but not defined", periodName)
		}

		// Calculate total delay for this period
		var totalDelay time.Duration
		for _, flowItem := range flow.Spec.Flows {
			if flowItem.PeriodName == periodName {
				for _, resource := range flowItem.Resources {
					if resource.Delay != nil {
						delay, err := time.ParseDuration(*resource.Delay)
						if err != nil {
							return fmt.Errorf("invalid delay format for resource %s: %w", resource.Name, err)
						}
						totalDelay += delay
					}
				}
			}
		}

		// Get period duration
		periodDuration, err := r.getPeriodDuration(&period)
		if err != nil {
			return fmt.Errorf("failed to get period duration for %s: %w", periodName, err)
		}

		// Check if total delay exceeds period duration
		if totalDelay > periodDuration {
			return fmt.Errorf("total delay %v for period %s exceeds period duration %v",
				totalDelay, periodName, periodDuration)
		}
	}

	return nil
}

// createResourceMappings creates mappings for all resources with their associated periods
//
//nolint:gocognit,gocyclo,gocritic // Resource mapping function complexity is acceptable for comprehensive processing
func (r *FlowReconciler) createResourceMappings(
	flow *kubecloudscalerv1alpha3.Flow,
	resourceNames map[string]bool,
) (map[string]ResourceInfo, error) {
	resourceMappings := make(map[string]ResourceInfo)

	// Create period map for lookup
	periodsMap := make(map[string]common.ScalerPeriod)
	for i := range flow.Spec.Periods {
		period := flow.Spec.Periods[i]
		periodName := ptr.Deref(period.Name, "")
		periodsMap[periodName] = period
	}

	// Process each resource name
	for resourceName := range resourceNames {
		// Find the resource in K8s resources
		var k8sResource *kubecloudscalerv1alpha3.K8sResource
		//nolint:gocritic // Range iteration of struct is acceptable, refactoring would reduce readability
		for _, resource := range flow.Spec.Resources.K8s {
			if resource.Name == resourceName {
				//nolint:gosec // Taking address of loop variable is safe here as it's used immediately
				k8sResource = &resource
				break
			}
		}

		// Find the resource in GCP resources
		var gcpResource *kubecloudscalerv1alpha3.GcpResource
		//nolint:gocritic // Range iteration of struct is acceptable, refactoring would reduce readability
		for _, resource := range flow.Spec.Resources.Gcp {
			if resource.Name == resourceName {
				//nolint:gosec // Taking address of loop variable is safe here as it's used immediately
				gcpResource = &resource
				break
			}
		}

		// Determine resource type and validate uniqueness
		var resourceType string
		var resourceObj interface{}
		if k8sResource != nil && gcpResource != nil {
			return nil, fmt.Errorf("resource %s is defined in both K8s and GCP resources", resourceName)
		} else if k8sResource != nil {
			resourceType = "k8s"
			resourceObj = *k8sResource
		} else if gcpResource != nil {
			resourceType = "gcp"
			resourceObj = *gcpResource
		} else {
			return nil, fmt.Errorf("resource %s referenced in flows but not defined in resources", resourceName)
		}

		// Find all periods associated with this resource
		var periodsWithDelay []PeriodWithDelay
		for _, flowItem := range flow.Spec.Flows {
			for _, resource := range flowItem.Resources {
				//nolint:gocritic // Nesting structure represents domain logic, inverting would reduce clarity
				if resource.Name == resourceName {
					period, exists := periodsMap[flowItem.PeriodName]
					if !exists {
						return nil, fmt.Errorf("period %s referenced in flows but not defined", flowItem.PeriodName)
					}

					// Calculate delay
					var delay time.Duration
					if resource.Delay != nil {
						parsedDelay, err := time.ParseDuration(*resource.Delay)
						if err != nil {
							return nil, fmt.Errorf("invalid delay format for resource %s: %w", resource.Name, err)
						}
						delay = parsedDelay
					}

					// Calculate start time (period start + delay)
					startTime, err := r.calculatePeriodStartTime(&period, delay)
					if err != nil {
						return nil, fmt.Errorf("failed to calculate start time for period %s: %w", flowItem.PeriodName, err)
					}
					startTime = startTime.Add(delay)

					periodsWithDelay = append(periodsWithDelay, PeriodWithDelay{
						Period:    period,
						Delay:     delay,
						StartTime: startTime,
					})
				}
			}
		}

		resourceMappings[resourceName] = ResourceInfo{
			Type:     resourceType,
			Resource: resourceObj,
			Periods:  periodsWithDelay,
		}
	}

	return resourceMappings, nil
}

// processResource processes a single resource and creates the appropriate CR
func (r *FlowReconciler) processResource(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	resourceInfo ResourceInfo,
) error {
	switch resourceInfo.Type {
	case "k8s":
		k8sResource, ok := resourceInfo.Resource.(kubecloudscalerv1alpha3.K8sResource)
		if !ok {
			return fmt.Errorf("expected K8sResource, got %T", resourceInfo.Resource)
		}
		return r.createK8sResource(ctx, flow, resourceName, k8sResource, resourceInfo.Periods)
	case "gcp":
		gcpResource, ok := resourceInfo.Resource.(kubecloudscalerv1alpha3.GcpResource)
		if !ok {
			return fmt.Errorf("expected GcpResource, got %T", resourceInfo.Resource)
		}
		return r.createGcpResource(ctx, flow, resourceName, gcpResource, resourceInfo.Periods)
	default:
		return fmt.Errorf("unknown resource type: %s", resourceInfo.Type)
	}
}

// calculatePeriodStartTime calculates the start time for a period with delay
func (r *FlowReconciler) calculatePeriodStartTime(period *common.ScalerPeriod, delay time.Duration) (time.Time, error) {
	if period.Time.Recurring != nil {
		// For recurring periods, parse the start time
		startTime, err := time.Parse("15:04", period.Time.Recurring.StartTime)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse recurring start time: %w", err)
		}
		// Add the delay to the start time
		startTime = startTime.Add(delay)
		return startTime, nil
	}

	if period.Time.Fixed != nil {
		// For fixed periods, parse the start time
		startTime, err := time.Parse("2006-01-02 15:04:05", period.Time.Fixed.StartTime)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse fixed start time: %w", err)
		}
		// Add the delay to the start time
		startTime = startTime.Add(delay)
		return startTime, nil
	}

	return time.Time{}, fmt.Errorf("no valid time period found")
}

// getPeriodDuration calculates the duration of a period
func (r *FlowReconciler) getPeriodDuration(period *common.ScalerPeriod) (time.Duration, error) {
	if period.Time.Recurring != nil {
		startTime, err := time.Parse("15:04", period.Time.Recurring.StartTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse start time: %w", err)
		}
		endTime, err := time.Parse("15:04", period.Time.Recurring.EndTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse end time: %w", err)
		}

		// Handle case where end time is before start time (next day)
		if endTime.Before(startTime) {
			return 0, fmt.Errorf("end time is before start time")
		}

		return endTime.Sub(startTime), nil
	}

	if period.Time.Fixed != nil {
		startTime, err := time.Parse("2006-01-02 15:04:05", period.Time.Fixed.StartTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse start time: %w", err)
		}
		endTime, err := time.Parse("2006-01-02 15:04:05", period.Time.Fixed.EndTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse end time: %w", err)
		}

		return endTime.Sub(startTime), nil
	}

	return 0, fmt.Errorf("no valid time period found")
}

// createK8sResource creates a K8s resource CR with all associated periods
//
//nolint:gocritic // Passing struct by value is acceptable for clarity and avoiding unintended mutations
func (r *FlowReconciler) createK8sResource(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	k8sResource kubecloudscalerv1alpha3.K8sResource,
	periodsWithDelay []PeriodWithDelay,
) error {
	// Collect all periods for this resource
	allPeriods := make([]common.ScalerPeriod, 0, len(periodsWithDelay))
	for _, periodWithDelay := range periodsWithDelay {
		curPeriod := periodWithDelay.Period
		if curPeriod.Time.Recurring != nil {
			curPeriod.Time.Recurring.StartTime = periodWithDelay.StartTime.Format("15:04")
		}
		if curPeriod.Time.Fixed != nil {
			curPeriod.Time.Fixed.StartTime = periodWithDelay.StartTime.Format("2006-01-02 15:04:05")
		}
		allPeriods = append(allPeriods, curPeriod)
	}

	// Create K8s object
	k8sObj := &kubecloudscalerv1alpha3.K8s{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("flow-%s-%s", flow.Name, resourceName),
			Labels: map[string]string{
				"flow":     flow.Name,
				"resource": resourceName,
			},
		},
		Spec: kubecloudscalerv1alpha3.K8sSpec{
			DryRun:    false,
			Periods:   allPeriods,
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
		Int("periods", len(allPeriods)).
		Msg("created/updated K8s resource")

	return nil
}

// createGcpResource creates a GCP resource CR with all associated periods
//
//nolint:gocritic // Passing struct by value is acceptable for clarity and avoiding unintended mutations
func (r *FlowReconciler) createGcpResource(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	gcpResource kubecloudscalerv1alpha3.GcpResource,
	periodsWithDelay []PeriodWithDelay,
) error {
	// Collect all periods for this resource
	allPeriods := make([]common.ScalerPeriod, 0, len(periodsWithDelay))
	for _, periodWithDelay := range periodsWithDelay {
		allPeriods = append(allPeriods, periodWithDelay.Period)
	}

	// Create GCP object
	gcpObj := &kubecloudscalerv1alpha3.Gcp{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("flow-%s-%s", flow.Name, resourceName),
			Labels: map[string]string{
				"flow":     flow.Name,
				"resource": resourceName,
			},
		},
		Spec: kubecloudscalerv1alpha3.GcpSpec{
			DryRun:    false,
			Periods:   allPeriods,
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
		Int("periods", len(allPeriods)).
		Msg("created/updated GCP resource")

	return nil
}

// updateFlowStatus updates the flow status with the given condition
//
//nolint:gocritic // Passing metav1.Condition by value is idiomatic in Kubernetes
func (r *FlowReconciler) updateFlowStatus(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	condition metav1.Condition,
) (ctrl.Result, error) {
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
