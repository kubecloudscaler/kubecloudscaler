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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
			latest, err := patchAddFlowFinalizer(ctx)
			switch {
			case apierrors.IsNotFound(err):
				// Flow was deleted between FetchHandler and this patch — nothing to do.
				ctx.SkipRemaining = true
				return nil
			case err != nil:
				ctx.RequeueAfter = shared.TransientRequeueAfter
				return shared.NewRecoverableError(fmt.Errorf("add finalizer: %w", err))
			}
			// Refresh ctx.Flow so ResourceVersion and finalizers match the persisted state;
			// downstream handlers that read ctx.Flow (e.g. for Update) must see the latest.
			ctx.Flow = latest
		}
		if h.next != nil && !ctx.SkipRemaining {
			return h.next.Execute(ctx)
		}
		return nil
	}

	if controllerutil.ContainsFinalizer(flow, flowFinalizer) {
		ctx.Logger.Info().Msg("removing finalizer")
		latest, err := patchRemoveFlowFinalizer(ctx)
		switch {
		case apierrors.IsNotFound(err):
			// Flow has already been fully deleted — cleanup is done.
			ctx.SkipRemaining = true
			return nil
		case err != nil:
			ctx.RequeueAfter = shared.TransientRequeueAfter
			return shared.NewRecoverableError(fmt.Errorf("remove finalizer: %w", err))
		}
		if latest != nil {
			ctx.Flow = latest
		}
	}
	ctx.SkipRemaining = true
	return nil
}

func (h *FinalizerHandler) SetNext(next service.Handler) {
	h.next = next
}

// patchAddFlowFinalizer adds flowFinalizer via an optimistic-locked merge patch, re-fetching
// and retrying on 409 conflicts. Scoped to metadata.finalizers so neither spec nor status is
// transmitted. Returns the latest persisted Flow so callers can refresh their in-memory copy.
// No-op if the finalizer was already present.
func patchAddFlowFinalizer(ctx *service.FlowReconciliationContext) (*kubecloudscalerv1alpha3.Flow, error) {
	latest := &kubecloudscalerv1alpha3.Flow{}
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest = &kubecloudscalerv1alpha3.Flow{}
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
	if err != nil {
		return nil, err
	}
	return latest, nil
}

// patchRemoveFlowFinalizer removes flowFinalizer via an optimistic-locked merge patch,
// re-fetching and retrying on 409 conflicts. Returns the latest persisted Flow when the
// finalizer was present and has been removed, nil when already absent. No-op if already absent.
func patchRemoveFlowFinalizer(ctx *service.FlowReconciliationContext) (*kubecloudscalerv1alpha3.Flow, error) {
	var result *kubecloudscalerv1alpha3.Flow
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.Flow{}
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, latest); err != nil {
			return err
		}
		if !controllerutil.ContainsFinalizer(latest, flowFinalizer) {
			result = nil
			return nil
		}
		patch := client.MergeFromWithOptions(latest.DeepCopy(), client.MergeFromWithOptimisticLock{})
		controllerutil.RemoveFinalizer(latest, flowFinalizer)
		if err := ctx.Client.Patch(ctx.Ctx, latest, patch); err != nil {
			return err
		}
		result = latest
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
