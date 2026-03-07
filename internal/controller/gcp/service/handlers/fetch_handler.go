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
//   - Return nil if resource is not found (e.g., owned by a Flow that was just removed)
//   - Return recoverable error for transient API failures
//
// Error Handling:
//   - Resource not found: Return nil (stops chain gracefully, no error logged)
//   - API errors: Set requeue and return nil (stops chain, allows retry)
type FetchHandler struct {
	next service.Handler
}

// NewFetchHandler creates a new fetch handler.
func NewFetchHandler() service.Handler {
	return &FetchHandler{}
}

// Execute implements the Handler interface.
// It fetches the scaler resource from the Kubernetes API and populates the context.
func (h *FetchHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := &kubecloudscalerv1alpha3.Gcp{}
	if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, scaler); err != nil {
		if client.IgnoreNotFound(err) == nil {
			ctx.SkipRemaining = true
			return nil
		}
		ctx.Logger.Warn().Err(err).Msg("transient error fetching scaler resource")
		ctx.RequeueAfter = transientRequeueAfter
		return service.NewRecoverableError(fmt.Errorf("fetch scaler resource: %w", err))
	}

	ctx.Scaler = scaler
	if h.next != nil {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *FetchHandler) SetNext(next service.Handler) {
	h.next = next
}
