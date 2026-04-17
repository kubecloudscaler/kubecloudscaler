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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/testutil"
)

var _ = Describe("PeriodHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		scaler   *kubecloudscalerv1alpha3.K8s
	)

	BeforeEach(func() {
		handler = handlers.NewPeriodHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		// Period spans every day across the full 24h window so it is always active
		// regardless of when the test runs — eliminates time-based non-determinism.
		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				Periods: []common.ScalerPeriod{
					{
						Name: "test-period",
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
				},
			},
		}

		reconCtx = &service.ReconciliationContext{
			Ctx: context.Background(),
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-scaler",
					Namespace: "default",
				},
			},
			Client:    fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build(),
			Logger:    &logger,
			Scaler:    scaler,
			K8sClient: fake.NewSimpleClientset(), // Mock K8s client
		}
	})

	Context("When an always-active period is configured", func() {
		It("should resolve the period and chain to the next handler", func() {
			nextCalled := false
			handler.SetNext(&testutil.MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			})

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Period).ToNot(BeNil())
			Expect(reconCtx.Period.Name).To(Equal("test-period"))
			Expect(reconCtx.SkipRemaining).To(BeFalse())
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("When noaction period is detected", func() {
		BeforeEach(func() {
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name: "noaction",
				Type: "noaction",
			}
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
		})

		It("should set SkipRemaining when current period is noaction", func() {
			nextCalled := false
			mockNext := &testutil.MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(nextCalled).To(BeFalse())
		})

		It("should NOT skip remaining when ShouldFinalize is true", func() {
			reconCtx.ShouldFinalize = true

			nextCalled := false
			mockNext := &testutil.MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeFalse())
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("When the previous period name collides with 'noaction' but its Type is not", func() {
		It("should not skip remaining (Type-based comparison, not Name)", func() {
			// A user legitimately creates a period literally named "noaction" but typed "down".
			// The system must not confuse it for the fallback noaction period.
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name: "noaction",
				Type: string(common.PeriodTypeDown),
			}
			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "noaction",
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

			nextCalled := false
			handler.SetNext(&testutil.MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			})

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeFalse())
			Expect(nextCalled).To(BeTrue())
		})
	})
})
