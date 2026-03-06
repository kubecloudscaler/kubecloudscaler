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
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/metrics"
)

// ScalerReconciler reconciles a GCP Scaler object.
// It manages the lifecycle of GCP resources by scaling them up/down based on configured periods.
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger *zerolog.Logger
	chain  service.Handler
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=gcps/finalizers,verbs=update

// Reconcile handles the scaling of GCP resources based on configured time periods.
//
// The reconciliation is handled by a Chain of Responsibility pattern:
//  1. FetchHandler — Fetch scaler resource from Kubernetes API
//  2. FinalizerHandler — Manage finalizer lifecycle
//  3. AuthHandler — Setup GCP client with authentication
//  4. PeriodHandler — Validate and determine current time period
//  5. ScalingHandler — Scale GCP resources based on period
//  6. StatusHandler — Update scaler status with operation results
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	r.Logger.Debug().Str("name", req.Name).Str("namespace", req.Namespace).Msg("reconcile start")

	if r.chain == nil {
		r.chain = r.initializeChain()
	}

	reconCtx := &service.ReconciliationContext{
		Ctx:     ctx,
		Request: req,
		Client:  r.Client,
		Logger:  r.Logger,
	}

	err := r.chain.Execute(reconCtx)

	metrics.ReconcileDurationSeconds.WithLabelValues(metrics.ControllerGCP).Observe(time.Since(startTime).Seconds())

	if err != nil {
		if service.IsCriticalError(err) {
			metrics.ReconcileTotal.WithLabelValues(metrics.ControllerGCP, metrics.ResultCriticalError).Inc()
			r.Logger.Error().Err(err).Str("name", req.Name).Str("namespace", req.Namespace).Msg("reconcile failed (critical)")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if service.IsRecoverableError(err) {
			metrics.ReconcileTotal.WithLabelValues(metrics.ControllerGCP, metrics.ResultRecoverableError).Inc()
			requeue := utils.ReconcileErrorDuration
			if reconCtx.RequeueAfter > 0 {
				requeue = reconCtx.RequeueAfter
			}
			r.Logger.Warn().Err(err).Str("name", req.Name).Dur("requeue_after", requeue).Msg("reconcile failed (recoverable)")
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		metrics.ReconcileTotal.WithLabelValues(metrics.ControllerGCP, metrics.ResultError).Inc()
		r.Logger.Error().Err(err).Str("name", req.Name).Str("namespace", req.Namespace).Msg("reconcile failed")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	metrics.ReconcileTotal.WithLabelValues(metrics.ControllerGCP, metrics.ResultSuccess).Inc()
	requeueAfter := reconCtx.RequeueAfter
	if requeueAfter > 0 {
		r.Logger.Info().Str("name", req.Name).Str("namespace", req.Namespace).Dur("requeue_after", requeueAfter).Msg("reconcile ok")
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	r.Logger.Info().Str("name", req.Name).Str("namespace", req.Namespace).Msg("reconcile ok")
	return ctrl.Result{}, nil
}

// initializeChain creates and links the handler chain in fixed order.
func (r *ScalerReconciler) initializeChain() service.Handler {
	fetchHandler := handlers.NewFetchHandler()
	finalizerHandler := handlers.NewFinalizerHandler()
	authHandler := handlers.NewAuthHandler()
	periodHandler := handlers.NewPeriodHandler()
	scalingHandler := handlers.NewScalingHandler()
	statusHandler := handlers.NewStatusHandler()

	fetchHandler.SetNext(finalizerHandler)
	finalizerHandler.SetNext(authHandler)
	authHandler.SetNext(periodHandler)
	periodHandler.SetNext(scalingHandler)
	scalingHandler.SetNext(statusHandler)

	return fetchHandler
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.chain = r.initializeChain()

	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha3.Gcp{}).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Named("gcpScaler").
		Complete(r)
}
