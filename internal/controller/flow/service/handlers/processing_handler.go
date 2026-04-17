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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
)

// ProcessingHandler delegates flow processing to the FlowProcessor service.
// It validates the flow, maps resources, and creates K8s/GCP child resources.
//
// On processing error it populates ctx.Condition (Status=False, Reason classified) and
// ctx.ProcessingError, then returns nil so StatusHandler still runs and persists the
// failure condition. The controller inspects ctx.ProcessingError after the chain and
// classifies it as CriticalError (ValidationError) or RecoverableError (transient).
type ProcessingHandler struct {
	next      service.Handler
	processor service.FlowProcessor
}

// NewProcessingHandler creates a new ProcessingHandler with the given FlowProcessor.
func NewProcessingHandler(processor service.FlowProcessor) service.Handler {
	return &ProcessingHandler{processor: processor}
}

func (h *ProcessingHandler) Execute(ctx *service.FlowReconciliationContext) error {
	if err := h.processor.ProcessFlow(ctx.Ctx, ctx.Flow); err != nil {
		reason := "ProcessingFailed"
		if v, ok := service.AsValidationError(err); ok {
			reason = v.Reason
		}
		ctx.ProcessingError = err
		ctx.Condition = &metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: err.Error(),
		}
	} else {
		ctx.Condition = &metav1.Condition{
			Type:    "Processed",
			Status:  metav1.ConditionTrue,
			Reason:  "ProcessingSucceeded",
			Message: "Flow processed successfully",
		}
	}

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

func (h *ProcessingHandler) SetNext(next service.Handler) {
	h.next = next
}
