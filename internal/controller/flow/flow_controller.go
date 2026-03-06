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
	"time"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/metrics"
)

const flowFinalizer = "kubecloudscaler.cloud/flow-finalizer"

// FlowReconciler reconciles a Flow object
// It manages the lifecycle of K8s and GCP resources by creating and deploying them
// based on configured periods and flow definitions with timing delays
type FlowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger *zerolog.Logger

	// Services for clean architecture
	flowProcessor service.FlowProcessor
	statusUpdater service.StatusUpdater
}

// NewFlowReconciler creates a new FlowReconciler with all required services
func NewFlowReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	logger *zerolog.Logger,
) *FlowReconciler {
	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}
	// Create services
	timeCalculator := service.NewTimeCalculatorService(logger)
	flowValidator := service.NewFlowValidatorService(timeCalculator, logger)
	resourceMapper := service.NewResourceMapperService(timeCalculator, logger)
	resourceCreator := service.NewResourceCreatorService(client, scheme, logger)
	flowProcessor := service.NewFlowProcessorService(flowValidator, resourceMapper, resourceCreator, logger)
	statusUpdater := service.NewStatusUpdaterService(client, logger)

	return &FlowReconciler{
		Client:        client,
		Scheme:        scheme,
		Logger:        logger,
		flowProcessor: flowProcessor,
		statusUpdater: statusUpdater,
	}
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
	startTime := time.Now()

	flow, err := r.fetchFlow(ctx, req)
	if err != nil {
		r.observeReconcile(startTime, metrics.ResultError)
		return ctrl.Result{}, err
	}

	r.Logger.Debug().Str("name", flow.Name).Msg("reconcile start")

	// Handle finalizer management
	if stop, result, err := r.handleFinalizers(ctx, flow); stop {
		resultLabel := metrics.ResultSuccess
		if err != nil {
			resultLabel = metrics.ResultError
		}
		r.observeReconcile(startTime, resultLabel)
		return result, err
	}

	// Process flow and create resources
	if err := r.flowProcessor.ProcessFlow(ctx, flow); err != nil {
		r.observeReconcile(startTime, metrics.ResultError)
		return r.handleProcessingError(ctx, flow, err)
	}

	r.observeReconcile(startTime, metrics.ResultSuccess)
	// Update status to indicate successful processing
	return r.statusUpdater.UpdateFlowStatus(ctx, flow, metav1.Condition{
		Type:    "Processed",
		Status:  metav1.ConditionTrue,
		Reason:  "ProcessingSucceeded",
		Message: "Flow processed successfully",
	})
}

func (r *FlowReconciler) observeReconcile(startTime time.Time, result string) {
	metrics.ReconcileDurationSeconds.WithLabelValues(metrics.ControllerFlow).Observe(time.Since(startTime).Seconds())
	metrics.ReconcileTotal.WithLabelValues(metrics.ControllerFlow, result).Inc()
}

// fetchFlow fetches the Flow object from the Kubernetes API
func (r *FlowReconciler) fetchFlow(ctx context.Context, req ctrl.Request) (*kubecloudscalerv1alpha3.Flow, error) {
	flow := &kubecloudscalerv1alpha3.Flow{}
	if err := r.Get(ctx, req.NamespacedName, flow); err != nil {
		r.Logger.Error().Err(err).Str("name", req.Name).Msg("fetch Flow failed")
		return nil, client.IgnoreNotFound(err)
	}
	return flow, nil
}

// handleFinalizers handles finalizer management for proper cleanup.
// Returns (stop bool, result ctrl.Result, err error).
// stop=true means the caller should return immediately with the given result and error.
func (r *FlowReconciler) handleFinalizers(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow) (bool, ctrl.Result, error) {
	if flow.DeletionTimestamp.IsZero() {
		// Object is not being deleted - ensure finalizer is present
		if !controllerutil.ContainsFinalizer(flow, flowFinalizer) {
			controllerutil.AddFinalizer(flow, flowFinalizer)
			if err := r.Update(ctx, flow); err != nil {
				r.Logger.Error().Err(err).Str("name", flow.Name).Msg("add finalizer failed")
				return true, ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, err
			}
		}
		return false, ctrl.Result{}, nil
	}

	if controllerutil.ContainsFinalizer(flow, flowFinalizer) {
		controllerutil.RemoveFinalizer(flow, flowFinalizer)
		if err := r.Update(ctx, flow); err != nil {
			r.Logger.Error().Err(err).Str("name", flow.Name).Msg("remove finalizer failed")
			return true, ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, err
		}
		return true, ctrl.Result{}, nil
	}

	// Finalizer already removed, stop reconciliation
	return true, ctrl.Result{}, nil
}

// handleProcessingError handles processing errors and updates flow status accordingly.
func (r *FlowReconciler) handleProcessingError(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow, err error) (ctrl.Result, error) {
	r.Logger.Error().Err(err).Str("name", flow.Name).Msg("flow processing failed")
	return r.statusUpdater.UpdateFlowStatus(ctx, flow, metav1.Condition{
		Type:    "Processed",
		Status:  metav1.ConditionFalse,
		Reason:  "ProcessingFailed",
		Message: err.Error(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Flow{}).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Named("flow").
		Complete(r)
}
