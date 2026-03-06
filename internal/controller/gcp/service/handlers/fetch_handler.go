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

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchHandler fetches the scaler resource from the Kubernetes API.
// This is the first handler in the chain and populates the Scaler field in the context.
type FetchHandler struct {
	next service.Handler
}

// NewFetchHandler creates a new fetch handler.
func NewFetchHandler() service.Handler {
	return &FetchHandler{}
}

// Execute fetches the GCP scaler resource and populates the context.
func (h *FetchHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := &kubecloudscalerv1alpha3.Gcp{}
	if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, scaler); err != nil {
		if client.IgnoreNotFound(err) == nil {
			ctx.Logger.Error().Err(err).Str("name", ctx.Request.Name).Str("namespace", ctx.Request.Namespace).Msg("scaler not found")
			return service.NewCriticalError(fmt.Errorf("scaler resource not found: %w", err))
		}

		ctx.Logger.Warn().Err(err).Msg("transient error fetching scaler resource")
		ctx.RequeueAfter = service.TransientRequeueAfter
		return nil
	}

	ctx.Scaler = scaler
	ctx.Logger.Info().Str("name", ctx.Scaler.Name).Str("namespace", ctx.Scaler.Namespace).Msg("scaler fetched")

	if h.next != nil {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *FetchHandler) SetNext(next service.Handler) {
	h.next = next
}
