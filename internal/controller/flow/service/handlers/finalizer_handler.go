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
	"fmt"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const flowFinalizer = "kubecloudscaler.cloud/flow-finalizer"

// FinalizerHandler manages the finalizer lifecycle for Flow resources.
// It adds a finalizer on creation and removes it during deletion.
type FinalizerHandler struct {
	next service.Handler
}

// NewFinalizerHandler creates a new FinalizerHandler.
func NewFinalizerHandler() service.Handler {
	return &FinalizerHandler{}
}

func (h *FinalizerHandler) Execute(ctx *service.FlowReconciliationContext) error {
	flow := ctx.Flow

	if flow.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(flow, flowFinalizer) {
			ctx.Logger.Info().Msg("adding finalizer")
			controllerutil.AddFinalizer(flow, flowFinalizer)
			if err := ctx.Client.Update(ctx.Ctx, flow); err != nil {
				ctx.RequeueAfter = shared.TransientRequeueAfter
				return shared.NewRecoverableError(fmt.Errorf("add finalizer: %w", err))
			}
		}
		if h.next != nil && !ctx.SkipRemaining {
			return h.next.Execute(ctx)
		}
		return nil
	}

	if controllerutil.ContainsFinalizer(flow, flowFinalizer) {
		ctx.Logger.Info().Msg("deleting flow with finalizer")
		ctx.Logger.Info().Msg("removing finalizer")
		controllerutil.RemoveFinalizer(flow, flowFinalizer)
		if err := ctx.Client.Update(ctx.Ctx, flow); err != nil {
			ctx.RequeueAfter = shared.TransientRequeueAfter
			return shared.NewRecoverableError(fmt.Errorf("remove finalizer: %w", err))
		}
	}
	ctx.SkipRemaining = true
	return nil
}

func (h *FinalizerHandler) SetNext(next service.Handler) {
	h.next = next
}
