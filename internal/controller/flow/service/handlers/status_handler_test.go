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

package handlers_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/testutil"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

var _ = Describe("StatusHandler", func() {
	var (
		logger   zerolog.Logger
		reconCtx *service.FlowReconciliationContext
		updater  *testutil.MockStatusUpdater
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		updater = &testutil.MockStatusUpdater{}
		reconCtx = &service.FlowReconciliationContext{
			Ctx:    context.Background(),
			Logger: &logger,
			Flow:   &kubecloudscalerv1alpha3.Flow{ObjectMeta: metav1.ObjectMeta{Name: "test-flow"}},
		}
	})

	Context("When ProcessingHandler populated a failure condition", func() {
		It("persists the failure condition exactly as recorded", func() {
			reconCtx.Condition = &metav1.Condition{
				Type:    "Processed",
				Status:  metav1.ConditionFalse,
				Reason:  "UnknownPeriod",
				Message: "period foo referenced in flows but not defined",
			}
			h := handlers.NewStatusHandler(updater)

			Expect(h.Execute(reconCtx)).To(Succeed())
			Expect(updater.CallCount).To(Equal(1))
			Expect(updater.LastCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(updater.LastCondition.Reason).To(Equal("UnknownPeriod"))
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})
	})

	Context("When no condition was populated", func() {
		It("falls back to a default ProcessingSucceeded condition", func() {
			h := handlers.NewStatusHandler(updater)

			Expect(h.Execute(reconCtx)).To(Succeed())
			Expect(updater.LastCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(updater.LastCondition.Reason).To(Equal("ProcessingSucceeded"))
		})
	})

	Context("When the status update fails", func() {
		It("returns a RecoverableError", func() {
			updater.UpdateFlowStatusFunc = func(_ context.Context, _ *kubecloudscalerv1alpha3.Flow, _ metav1.Condition) error {
				return fmt.Errorf("boom")
			}
			h := handlers.NewStatusHandler(updater)

			err := h.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(shared.IsRecoverableError(err)).To(BeTrue())
		})
	})
})
