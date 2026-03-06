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
	"time"

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
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
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

	Context("When noaction period remains active", func() {
		BeforeEach(func() {
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "noaction",
					Type: "noaction",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "00:00",
							EndTime:   "23:59",
							Once:      ptr.To(false),
						},
					},
				},
			}
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{Name: "noaction"}

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
				GCPClient: &gcpUtils.ClientSet{}, // Mock GCP client
			}
		})

		It("should skip remaining handlers and requeue", func() {
			err := periodHandler.Execute(reconCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})
	})

	Context("When transitioning to noaction from active period", func() {
		BeforeEach(func() {
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "noaction",
					Type: "noaction",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "00:00",
							EndTime:   "23:59",
							Once:      ptr.To(false),
						},
					},
				},
			}
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{Name: "up"}

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

		It("should continue to next handler", func() {
			nextCalled := false
			periodHandler.SetNext(&mockGCPHandler{
				executeFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			})

			err := periodHandler.Execute(reconCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(time.Duration(0)))
			Expect(reconCtx.SkipRemaining).To(BeFalse())
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("When run-once period was already processed", func() {
		BeforeEach(func() {
			now := time.Now()
			start := now.Add(-time.Minute).Format(time.DateTime)
			end := now.Add(time.Minute).Format(time.DateTime)

			runOnce := common.ScalerPeriod{
				Name: "once-down",
				Type: "down",
				Time: common.TimePeriod{
					Fixed: &common.FixedPeriod{
						StartTime: start,
						EndTime:   end,
						Once:      ptr.To(true),
					},
				},
			}
			scaler.Spec.Periods = []common.ScalerPeriod{runOnce}

			curPeriod, err := periodPkg.New(&runOnce)
			Expect(err).ToNot(HaveOccurred())
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name:    runOnce.Name,
				Type:    runOnce.Type,
				SpecSHA: curPeriod.Hash,
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

		It("should return a requeue result and not execute next handler", func() {
			nextCalled := false
			periodHandler.SetNext(&mockGCPHandler{
				executeFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			})

			err := periodHandler.Execute(reconCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(BeNumerically(">", time.Second))
			Expect(nextCalled).To(BeFalse())
		})
	})

	Context("When period configuration is invalid", func() {
		BeforeEach(func() {
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "bad-day",
					Type: "up",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"invalid-day"},
							StartTime: "09:00",
							EndTime:   "17:00",
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
				Ctx:       context.Background(),
				Request:   ctrl.Request{},
				Client:    k8sClient,
				Logger:    &logger,
				Scaler:    scaler,
				GCPClient: &gcpUtils.ClientSet{},
			}
		})

		It("should return critical error", func() {
			err := periodHandler.Execute(reconCtx)
			Expect(reconCtx.RequeueAfter).To(Equal(time.Duration(0)))
			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
		})
	})
})

type mockGCPHandler struct {
	executeFunc func(ctx *service.ReconciliationContext) error
}

func (m *mockGCPHandler) Execute(ctx *service.ReconciliationContext) error {
	if m.executeFunc == nil {
		return nil
	}
	return m.executeFunc(ctx)
}

func (m *mockGCPHandler) SetNext(_ service.Handler) {}
