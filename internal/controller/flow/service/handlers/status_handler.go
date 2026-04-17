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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

var _ service.Handler = (*StatusHandler)(nil)

// StatusHandler updates the Flow status using the StatusUpdater service.
// It sets a success condition and configures requeue timing.
type StatusHandler struct {
	next          service.Handler
	statusUpdater service.StatusUpdater
}

// NewStatusHandler creates a new StatusHandler with the given StatusUpdater.
func NewStatusHandler(statusUpdater service.StatusUpdater) service.Handler {
	return &StatusHandler{statusUpdater: statusUpdater}
}

func (h *StatusHandler) Execute(ctx *service.FlowReconciliationContext) error {
	// Prefer the condition populated by ProcessingHandler so both success and failure paths
	// are reflected on Flow.status. Fall back to a default success condition only when no
	// handler contributed one (e.g. the chain terminated before ProcessingHandler for a
	// reason other than deletion cleanup).
	cond := ctx.Condition
	if cond == nil {
		cond = &metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionTrue,
			Reason:  "ProcessingSucceeded",
			Message: "Flow processed successfully",
		}
	}

	if err := h.statusUpdater.UpdateFlowStatus(ctx.Ctx, ctx.Flow, *cond); err != nil {
		return shared.NewRecoverableError(fmt.Errorf("update flow status: %w", err))
	}

	if ctx.RequeueAfter == 0 {
		ctx.RequeueAfter = utils.ReconcileSuccessDuration
	}

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

func (h *StatusHandler) SetNext(next service.Handler) {
	h.next = next
}
