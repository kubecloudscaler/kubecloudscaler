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

package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

// ReconciliationContext holds the shared state for the Chain of Responsibility.
// Each handler can read from and write to this context.
//
// Fields are populated by handlers as the chain executes:
//   - Initial: Request, Client, Logger set by controller
//   - After Fetch: Scaler set
//   - After Finalizer: ShouldFinalize may be set
//   - After Auth: K8sClient, DynamicClient, Secret set
//   - After Period: Period, ResourceConfig set
//   - After Scaling: SuccessResults, FailedResults set
//   - After Status: Status updated in cluster
//
// Context Modification Rules:
//   - Later handlers overwrite earlier changes (last write wins)
//   - First handler to set RequeueAfter takes precedence (earliest handler wins)
//   - Handlers can set SkipRemaining to stop chain early
//
// See contracts/reconciliation-context.md for full contract specification.
type ReconciliationContext struct {
	// Ctx is the Go context from the Reconcile method.
	// Set by: Controller (before chain execution)
	// Used by: All handlers for API calls
	Ctx context.Context

	// Request is the controller request with NamespacedName.
	// Set by: Controller (before chain execution)
	// Used by: All handlers
	Request ctrl.Request

	// Client is the Kubernetes client for API operations (fetching/updating resources).
	// Set by: Controller (before chain execution)
	// Used by: All handlers
	Client client.Client

	// K8sClient is the typed Kubernetes client for resource operations.
	// Set by: AuthHandler
	// Used by: PeriodHandler, ScalingHandler
	K8sClient kubernetes.Interface

	// DynamicClient is the dynamic Kubernetes client for resource operations.
	// Set by: AuthHandler
	// Used by: PeriodHandler, ScalingHandler
	DynamicClient dynamic.Interface

	// Logger is the structured logger for handler execution logging.
	// Set by: Controller (before chain execution)
	// Used by: All handlers
	Logger *zerolog.Logger

	// Scaler is the K8s scaler resource being reconciled.
	// Set by: FetchHandler
	// Used by: All subsequent handlers
	Scaler *kubecloudscalerv1alpha3.K8s

	// Secret is the authentication secret for remote cluster access (nil if not needed).
	// Set by: AuthHandler
	// Used by: AuthHandler only
	Secret *corev1.Secret

	// Period is the current time period for scaling operations.
	// Set by: PeriodHandler
	// Used by: ScalingHandler, StatusHandler
	Period *period.Period

	// ResourceConfig is the resource configuration for scaling operations.
	// Set by: PeriodHandler
	// Used by: ScalingHandler
	ResourceConfig resources.Config

	// SuccessResults contains successful scaling operations.
	// Set by: ScalingHandler
	// Used by: StatusHandler
	SuccessResults []common.ScalerStatusSuccess

	// FailedResults contains failed scaling operations.
	// Set by: ScalingHandler
	// Used by: StatusHandler
	FailedResults []common.ScalerStatusFailed

	// ShouldFinalize indicates finalizer cleanup is needed.
	// Set by: FinalizerHandler (when deletion detected)
	// Used by: StatusHandler
	ShouldFinalize bool

	// SkipRemaining indicates chain should stop early (no remaining handlers should execute).
	// Set by: Any handler (e.g., FinalizerHandler, PeriodHandler)
	// Used by: Handlers check this flag before calling next.Execute()
	SkipRemaining bool

	// RequeueAfter is the requeue delay duration (first handler to set wins).
	// Set by: Any handler (e.g., PeriodHandler for run-once periods)
	// Used by: Controller (uses this value in ctrl.Result)
	RequeueAfter time.Duration
}
