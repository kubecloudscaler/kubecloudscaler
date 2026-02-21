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

// Package handlers provides handler implementations for the Chain of Responsibility pattern.
package handlers

import (
	"fmt"
	"time"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// transientRequeueAfter is the delay before retrying after a transient error.
// It is intentionally shorter than utils.ReconcileErrorDuration (10m) because
// these errors (API blip, add/remove finalizer failure) are expected to resolve quickly.
const transientRequeueAfter = 5 * time.Second

// FetchHandler fetches the scaler resource from the Kubernetes API.
// This is the first handler in the chain and populates the Scaler field in the context.
//
// Responsibilities:
//   - Fetch the scaler resource using the Kubernetes client
//   - Populate the Scaler field in the ReconciliationContext
//   - Return critical error if resource is not found
//   - Return recoverable error for transient API failures
//
// Error Handling:
//   - Resource not found: Critical error (stops chain)
//   - API errors: Recoverable error (allows retry with requeue)
type FetchHandler struct {
	next service.Handler
}

// NewFetchHandler creates a new fetch handler.
func NewFetchHandler() service.Handler {
	return &FetchHandler{}
}

// Execute implements the Handler interface.
// It fetches the scaler resource from the Kubernetes API and populates the context.
func (h *FetchHandler) Execute(req *service.ReconciliationContext) (ctrl.Result, error) {
	req.Logger.Debug().Msg("fetching scaler resource")

	ctx := req.Ctx

	// Fetch the Scaler object from the Kubernetes API
	scaler := &kubecloudscalerv1alpha3.Gcp{}
	if err := req.Client.Get(ctx, req.Request.NamespacedName, scaler); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Resource not found - critical error (stops chain)
			req.Logger.Error().Err(err).Msg("scaler resource not found")
			return ctrl.Result{}, service.NewCriticalError(fmt.Errorf("scaler resource not found: %w", err))
		}

		// Other API error - stop chain and requeue (Scaler is nil, next handlers would panic)
		req.Logger.Warn().Err(err).Msg("transient error fetching scaler resource")
		return ctrl.Result{RequeueAfter: transientRequeueAfter}, nil
	}

	// Successfully fetched - populate context and continue
	req.Scaler = scaler
	req.Logger.Info().
		Str("name", scaler.Name).
		Str("namespace", scaler.Namespace).
		Msg("scaler resource fetched successfully")

	if h.next != nil {
		return h.next.Execute(req)
	}
	return ctrl.Result{}, nil
}

// SetNext sets the next handler in the chain.
func (h *FetchHandler) SetNext(next service.Handler) {
	h.next = next
}
