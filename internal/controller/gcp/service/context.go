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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

// ReconciliationContext is a container for state shared between handlers during reconciliation.
// Handlers modify this context directly to pass information to subsequent handlers.
//
// State Transitions:
//   - Initial: Empty context with Request and Client only
//   - After Fetch: Scaler populated
//   - After Finalizer: ShouldFinalize flag set if deletion in progress
//   - After Auth: GCPClient and Secret populated
//   - After Period: Period and ResourceConfig populated
//   - After Scaling: SuccessResults and FailedResults populated
//   - After Status: Context ready for next reconciliation cycle
//
// Validation Rules:
//   - Scaler must be non-nil after fetch handler
//   - GCPClient must be non-nil after auth handler (if authentication succeeds)
//   - Period must be valid after period validation handler
//
// See data-model.md for full specification.
type ReconciliationContext struct {
	// Ctx is the Go context from the Reconcile method.
	// Set by: Controller (before chain execution)
	// Used by: All handlers for API calls
	Ctx context.Context

	// Request is the reconciliation request (namespaced name)
	Request ctrl.Request

	// Client is the Kubernetes client for API operations
	Client client.Client

	// Logger is the structured logger for observability
	Logger *zerolog.Logger

	// Scaler is the GCP scaler resource being reconciled (populated by fetch handler)
	Scaler *kubecloudscalerv1alpha3.Gcp

	// Secret is the authentication secret for GCP access (populated by auth handler, nullable)
	Secret *corev1.Secret

	// GCPClient is the GCP Compute Engine API client (populated by auth handler)
	GCPClient *gcpUtils.ClientSet

	// Period is the current time period configuration (populated by period validation handler)
	Period *periodPkg.Period

	// ResourceConfig is the resource management configuration (populated by period validation handler)
	ResourceConfig resources.Config

	// SuccessResults contains successfully scaled resources (populated by scaling handler)
	SuccessResults []common.ScalerStatusSuccess

	// FailedResults contains failed scaling operations (populated by scaling handler)
	FailedResults []common.ScalerStatusFailed

	// ShouldFinalize indicates if finalizer cleanup is needed (set by finalizer handler)
	ShouldFinalize bool

	// SkipRemaining indicates if remaining handlers should be skipped (set by any handler)
	SkipRemaining bool
}

// ReconciliationResult encapsulates the outcome of handler execution.
// This result determines how the chain should proceed after the handler completes.
//
// Validation Rules:
//   - If Error is non-nil and ErrorCategory is Critical, Continue must be false
//   - If Requeue is true, RequeueAfter must be > 0
//   - If SkipRemaining is requested, Continue must be false
//
// See data-model.md for full specification.
type ReconciliationResult struct {
	// Continue indicates whether the chain should continue to the next handler
	// Set to false to stop chain execution (e.g., on critical error or skip request)
	Continue bool

	// Requeue indicates whether reconciliation should be requeued
	// Set to true for recoverable errors or temporary conditions (e.g., run-once period)
	Requeue bool

	// RequeueAfter is the delay before requeue (if Requeue is true)
	// Must be > 0 if Requeue is true
	RequeueAfter time.Duration

	// Error is the error encountered during handler execution
	// Nil if no error occurred
	Error error

	// ErrorCategory categorizes the error for appropriate handling
	// Critical errors stop the chain, recoverable errors allow continuation with retry
	ErrorCategory ErrorCategory
}
