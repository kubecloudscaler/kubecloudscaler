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
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchHandler fetches the Flow resource from the Kubernetes API server.
//
// On success: sets ctx.Flow and continues the chain.
// On NotFound: returns nil — the Flow was deleted; reconcile has nothing to do (no error log).
// On other errors: returns RecoverableError for requeue.
type FetchHandler struct {
	next service.Handler
}

// NewFetchHandler creates a new FetchHandler.
func NewFetchHandler() service.Handler {
	return &FetchHandler{}
}

func (h *FetchHandler) Execute(ctx *service.FlowReconciliationContext) error {
	flow := &kubecloudscalerv1alpha3.Flow{}
	if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, flow); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return shared.NewRecoverableError(err)
	}

	ctx.Flow = flow
	ctx.Logger.Info().
		Str("name", flow.Name).
		Str("kind", flow.Kind).
		Str("apiVersion", flow.APIVersion).
		Msg("reconciling flow")

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

func (h *FetchHandler) SetNext(next service.Handler) {
	h.next = next
}
