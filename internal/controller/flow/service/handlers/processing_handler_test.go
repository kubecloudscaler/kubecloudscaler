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
)

// stubProcessor returns a preset error or nil from ProcessFlow.
type stubProcessor struct {
	err error
}

func (s *stubProcessor) ProcessFlow(_ context.Context, _ *kubecloudscalerv1alpha3.Flow) error {
	return s.err
}

var _ = Describe("ProcessingHandler", func() {
	var (
		logger   zerolog.Logger
		reconCtx *service.FlowReconciliationContext
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		reconCtx = &service.FlowReconciliationContext{
			Ctx:    context.Background(),
			Logger: &logger,
			Flow:   &kubecloudscalerv1alpha3.Flow{ObjectMeta: metav1.ObjectMeta{Name: "test-flow"}},
		}
	})

	Context("When ProcessFlow succeeds", func() {
		It("populates a success condition and leaves ProcessingError nil", func() {
			h := handlers.NewProcessingHandler(&stubProcessor{})

			Expect(h.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.ProcessingError).ToNot(HaveOccurred())
			Expect(reconCtx.Condition).ToNot(BeNil())
			Expect(reconCtx.Condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(reconCtx.Condition.Reason).To(Equal("ProcessingSucceeded"))
		})
	})

	Context("When ProcessFlow returns a ValidationError", func() {
		It("populates a failure condition with the validation Reason and stores ProcessingError", func() {
			h := handlers.NewProcessingHandler(&stubProcessor{
				err: service.NewValidationError("UnknownPeriod",
					fmt.Errorf("period foo referenced in flows but not defined")),
			})

			Expect(h.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.ProcessingError).To(HaveOccurred())
			Expect(service.IsValidationError(reconCtx.ProcessingError)).To(BeTrue())
			Expect(reconCtx.Condition).ToNot(BeNil())
			Expect(reconCtx.Condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(reconCtx.Condition.Reason).To(Equal("UnknownPeriod"))
		})
	})

	Context("When ProcessFlow returns a transient error", func() {
		It("populates a generic failure condition and stores ProcessingError", func() {
			h := handlers.NewProcessingHandler(&stubProcessor{err: fmt.Errorf("API down")})

			Expect(h.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.ProcessingError).To(HaveOccurred())
			Expect(service.IsValidationError(reconCtx.ProcessingError)).To(BeFalse())
			Expect(reconCtx.Condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(reconCtx.Condition.Reason).To(Equal("ProcessingFailed"))
		})
	})
})
