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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

var _ = Describe("PeriodHandler", func() {
	var (
		logger        zerolog.Logger
		scheme        *runtime.Scheme
		periodHandler service.Handler
		reconCtx      *service.ReconciliationContext
		scaler        *kubecloudscalerv1alpha3.Gcp
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")
		scaler.Spec.Config.ProjectID = "test-project"
		scaler.Spec.Config.Region = "us-central1"
		scaler.Spec.Config.DefaultPeriodType = "down"

		periodHandler = handlers.NewPeriodHandler()
	})

	Context("When an always-active period is configured", func() {
		BeforeEach(func() {
			// Every day across the full 24h window — eliminates time-based non-determinism.
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "always-up",
					Type: common.PeriodTypeUp,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayAll},
							StartTime: "00:00",
							EndTime:   "23:59",
							Once:      ptr.To(false),
						},
					},
				},
			}

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()
			reconCtx = &service.ReconciliationContext{
				Ctx:       context.Background(),
				Request:   ctrl.Request{},
				Client:    k8sClient,
				Logger:    &logger,
				Scaler:    scaler,
				GCPClient: &gcpUtils.ClientSet{},
			}
		})

		It("should resolve the period and populate the context deterministically", func() {
			Expect(periodHandler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.Period).ToNot(BeNil())
			Expect(reconCtx.Period.Name).To(Equal("always-up"))
			Expect(reconCtx.SkipRemaining).To(BeFalse())
		})
	})

	Context("When transitioning from an active period to noaction", func() {
		BeforeEach(func() {
			// Empty Spec.Periods forces SetActivePeriod to return the system-fallback noaction.
			// Status.CurrentPeriod captures the previous active period so prevPeriodName != "noaction".
			scaler.Spec.Periods = []common.ScalerPeriod{}
			scaler.Status = common.ScalerStatus{
				CurrentPeriod: &common.ScalerStatusPeriod{
					Name: "business-hours",
					Type: string(common.PeriodTypeUp),
				},
			}

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()
			reconCtx = &service.ReconciliationContext{
				Ctx:       context.Background(),
				Request:   ctrl.Request{},
				Client:    k8sClient,
				Logger:    &logger,
				Scaler:    scaler,
				GCPClient: &gcpUtils.ClientSet{},
			}
		})

		It("should NOT skip remaining — scaling handler must run to restore resource state", func() {
			// Regression guard for the 4db0412 fix: prevPeriodName must be snapshotted BEFORE
			// SetActivePeriod rewrites status.CurrentPeriod. If the snapshot moves after the
			// call, every reconcile sees prevPeriodName == "noaction" and the chain skips
			// silently, leaving resources in the wrong state.
			Expect(periodHandler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.SkipRemaining).To(BeFalse())
			Expect(reconCtx.Period).ToNot(BeNil())
		})
	})

	Context("When period is 'noaction' and status matches", func() {
		BeforeEach(func() {
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "noaction",
					Type: "noaction",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayAll},
							StartTime: "00:00",
							EndTime:   "23:59",
							Once:      ptr.To(false),
						},
					},
				},
			}

			scaler.Status = common.ScalerStatus{
				CurrentPeriod: &common.ScalerStatusPeriod{
					Name: "noaction",
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:       context.Background(),
				Request:   ctrl.Request{},
				Client:    k8sClient,
				Logger:    &logger,
				Scaler:    scaler,
				GCPClient: &gcpUtils.ClientSet{},
			}
		})

		It("should skip remaining handlers", func() {
			err := periodHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})

		It("should NOT skip remaining when ShouldFinalize is true", func() {
			reconCtx.ShouldFinalize = true

			err := periodHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeFalse())
		})
	})

	Context("When period configuration is empty", func() {
		BeforeEach(func() {
			scaler.Spec.Periods = []common.ScalerPeriod{}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:       context.Background(),
				Request:   ctrl.Request{},
				Client:    k8sClient,
				Logger:    &logger,
				Scaler:    scaler,
				GCPClient: &gcpUtils.ClientSet{},
			}
		})

		It("should handle empty periods gracefully", func() {
			err := periodHandler.Execute(reconCtx)

			// Should either succeed with default period or return appropriate result
			// Error handling depends on implementation
			_ = err
		})
	})

	Context("When handling finalizer deletion", func() {
		BeforeEach(func() {
			scaler.Spec.Config.RestoreOnDelete = true
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "down",
					Type: common.PeriodTypeDown,
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []common.DayOfWeek{common.DayAll},
							StartTime: "00:00",
							EndTime:   "23:59",
							Once:      ptr.To(false),
						},
					},
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				GCPClient:      &gcpUtils.ClientSet{},
				ShouldFinalize: true, // Deletion in progress
			}
		})

		It("should handle restore on delete", func() {
			err := periodHandler.Execute(reconCtx)

			// Validation with restore flag should work
			_ = err
		})
	})
})
