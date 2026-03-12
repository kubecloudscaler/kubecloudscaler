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
	"sync"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
	"github.com/kubecloudscaler/kubecloudscaler/internal/metrics"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// FlowReconciler reconciles a Flow object.
// It manages the lifecycle of K8s and GCP resources by creating and deploying them
// based on configured periods and flow definitions with timing delays.
//
// This controller uses the Chain of Responsibility pattern following the classic
// refactoring.guru Go pattern where handlers have Execute() and SetNext() methods.
type FlowReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Logger   *zerolog.Logger
	recorder metrics.Recorder

	chain     service.Handler
	chainOnce sync.Once

	flowProcessor service.FlowProcessor
	statusUpdater service.StatusUpdater
}

// NewFlowReconciler creates a new FlowReconciler with all required services.
func NewFlowReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	logger *zerolog.Logger,
) *FlowReconciler {
	if logger == nil {
		nopLogger := zerolog.Nop()
		logger = &nopLogger
	}

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
		recorder:      metrics.GetRecorder(),
		flowProcessor: flowProcessor,
		statusUpdater: statusUpdater,
	}
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=flows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=flows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=flows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// The reconciliation process is implemented as a Chain of Responsibility:
// 1. FetchHandler: Fetches the Flow object from the cluster
// 2. FinalizerHandler: Manages finalizers for proper cleanup
// 3. ProcessingHandler: Validates flow and creates K8s/GCP child resources
// 4. StatusHandler: Updates the status with results
func (r *FlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	rec := r.recorder
	if rec == nil {
		rec = metrics.GetRecorder()
	}

	reconCtx := &service.FlowReconciliationContext{
		Ctx:     ctx,
		Request: req,
		Client:  r.Client,
		Logger:  r.Logger,
	}

	r.chainOnce.Do(func() {
		if r.chain == nil {
			r.chain = r.initializeChain()
		}
	})

	err := r.chain.Execute(reconCtx)
	duration := time.Since(start).Seconds()

	if err != nil {
		if shared.IsCriticalError(err) {
			rec.RecordReconcile(metrics.ControllerFlow, metrics.ResultCriticalError, duration)
			r.Logger.Error().Err(err).Msg("critical error during reconciliation")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if shared.IsRecoverableError(err) {
			rec.RecordReconcile(metrics.ControllerFlow, metrics.ResultRecoverableError, duration)
			r.Logger.Warn().Err(err).Msg("recoverable error during reconciliation, will requeue")
			requeue := utils.ReconcileErrorDuration
			if reconCtx.RequeueAfter > 0 {
				requeue = reconCtx.RequeueAfter
			}
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		rec.RecordReconcile(metrics.ControllerFlow, metrics.ResultRecoverableError, duration)
		r.Logger.Error().Err(err).Msg("unexpected error during reconciliation")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	rec.RecordReconcile(metrics.ControllerFlow, metrics.ResultSuccess, duration)

	if reconCtx.RequeueAfter > 0 {
		return ctrl.Result{RequeueAfter: reconCtx.RequeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

// initializeChain creates and links the handler chain in fixed order.
//
// Handler Order:
// 1. FetchHandler → 2. FinalizerHandler → 3. ProcessingHandler → 4. StatusHandler
func (r *FlowReconciler) initializeChain() service.Handler {
	fetchHandler := handlers.NewFetchHandler()
	finalizerHandler := handlers.NewFinalizerHandler()
	processingHandler := handlers.NewProcessingHandler(r.flowProcessor)
	statusHandler := handlers.NewStatusHandler(r.statusUpdater)

	fetchHandler.SetNext(finalizerHandler)
	finalizerHandler.SetNext(processingHandler)
	processingHandler.SetNext(statusHandler)

	return fetchHandler
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.chain = r.initializeChain()

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Flow{}).
		Owns(&kubecloudscalerv1alpha3.K8s{}).
		Owns(&kubecloudscalerv1alpha3.Gcp{}).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Named("flow").
		Complete(r)
}
