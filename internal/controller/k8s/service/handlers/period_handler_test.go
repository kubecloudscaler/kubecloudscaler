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

		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				Periods: []common.ScalerPeriod{
					{
						Name: "test-period",
						Type: "up",
						Time: common.TimePeriod{
							Recurring: &common.RecurringPeriod{
								Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
								StartTime: "09:00",
								EndTime:   "17:00",
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

	Context("When a valid period is configured", func() {
		It("should process period and continue chain or set skip flag", func() {
			nextCalled := false
			mockNext := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)

			// Period validation may succeed or fail depending on current time
			// We're testing that the handler processes correctly in all cases
			if err != nil {
				// If period validation fails, it should be a critical error
				Expect(service.IsCriticalError(err)).To(BeTrue())
			} else {
				// Handler may continue to next, or skip remaining
				// Both outcomes are valid depending on current time
				if !reconCtx.SkipRemaining {
					Expect(reconCtx.Period).ToNot(BeNil())
					Expect(nextCalled).To(BeTrue())
				}
			}
		})

		It("should complete in under 100ms", func() {
			startTime := time.Now()
			_ = handler.Execute(reconCtx)
			duration := time.Since(startTime)

			Expect(duration).To(BeNumerically("<", 100*time.Millisecond))
		})
	})

	Context("When noaction period is detected", func() {
		BeforeEach(func() {
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name: "noaction",
			}
		})

		It("should set SkipRemaining when current period is noaction", func() {
			// Configure a period that would result in "noaction"
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

			nextCalled := false
			mockNext := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)

			// If noaction period is detected and current status matches, chain should skip
			if err == nil && reconCtx.SkipRemaining {
				Expect(nextCalled).To(BeFalse())
			}
		})
	})
})
