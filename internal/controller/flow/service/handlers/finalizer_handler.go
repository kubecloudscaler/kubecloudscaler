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

	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
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
			if err := patchAddFlowFinalizer(ctx); err != nil {
				ctx.RequeueAfter = shared.TransientRequeueAfter
				return shared.NewRecoverableError(fmt.Errorf("add finalizer: %w", err))
			}
			controllerutil.AddFinalizer(flow, flowFinalizer)
		}
		if h.next != nil && !ctx.SkipRemaining {
			return h.next.Execute(ctx)
		}
		return nil
	}

	if controllerutil.ContainsFinalizer(flow, flowFinalizer) {
		ctx.Logger.Info().Msg("removing finalizer")
		if err := patchRemoveFlowFinalizer(ctx); err != nil {
			ctx.RequeueAfter = shared.TransientRequeueAfter
			return shared.NewRecoverableError(fmt.Errorf("remove finalizer: %w", err))
		}
		controllerutil.RemoveFinalizer(flow, flowFinalizer)
	}
	ctx.SkipRemaining = true
	return nil
}

func (h *FinalizerHandler) SetNext(next service.Handler) {
	h.next = next
}

// patchAddFlowFinalizer adds flowFinalizer via an optimistic-locked merge patch, re-fetching
// and retrying on 409 conflicts. Scoped to metadata.finalizers so neither spec nor status is
// transmitted. No-op if the finalizer was added concurrently.
func patchAddFlowFinalizer(ctx *service.FlowReconciliationContext) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.Flow{}
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, latest); err != nil {
			return err
		}
		if controllerutil.ContainsFinalizer(latest, flowFinalizer) {
			return nil
		}
		patch := client.MergeFromWithOptions(latest.DeepCopy(), client.MergeFromWithOptimisticLock{})
		controllerutil.AddFinalizer(latest, flowFinalizer)
		return ctx.Client.Patch(ctx.Ctx, latest, patch)
	})
}

// patchRemoveFlowFinalizer removes flowFinalizer via an optimistic-locked merge patch,
// re-fetching and retrying on 409 conflicts. No-op if already absent.
func patchRemoveFlowFinalizer(ctx *service.FlowReconciliationContext) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.Flow{}
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, latest); err != nil {
			return err
		}
		if !controllerutil.ContainsFinalizer(latest, flowFinalizer) {
			return nil
		}
		patch := client.MergeFromWithOptions(latest.DeepCopy(), client.MergeFromWithOptimisticLock{})
		controllerutil.RemoveFinalizer(latest, flowFinalizer)
		return ctx.Client.Patch(ctx.Ctx, latest, patch)
	})
}
