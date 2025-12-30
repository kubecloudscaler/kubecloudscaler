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

package handlers

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
)

// FetchHandler is a handler that fetches the Scaler object from the Kubernetes API.
// This is the first handler in the chain and must set ctx.Scaler before other handlers can process.
type FetchHandler struct {
	next service.Handler
}

// NewFetchHandler creates a new FetchHandler.
func NewFetchHandler() service.Handler {
	return &FetchHandler{}
}

// Execute fetches the Scaler object and adds it to the reconciliation context.
//
// On success: Sets ctx.Scaler and calls next.Execute()
// On error: Returns CriticalError for not found, RecoverableError for transient errors
func (h *FetchHandler) Execute(ctx *service.ReconciliationContext) error {
	ctx.Logger.Debug().Msg("fetching scaler resource")

	scaler := &kubecloudscalerv1alpha3.K8s{}
	if err := ctx.Client.Get(context.Background(), ctx.Request.NamespacedName, scaler); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ctx.Logger.Error().Err(err).Msg("unable to fetch Scaler")
			return service.NewRecoverableError(err)
		}
		// If not found, it's a critical error for this chain, as we can't proceed without the scaler.
		return service.NewCriticalError(errors.New("scaler resource not found"))
	}

	ctx.Scaler = scaler
	ctx.Logger.Info().Str("name", scaler.Name).Str("namespace", scaler.Namespace).Msg("scaler resource fetched successfully")

	// Call next handler in chain
	if h.next != nil {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *FetchHandler) SetNext(next service.Handler) {
	h.next = next
}
