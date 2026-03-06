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
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
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

	Context("When noaction period remains active", func() {
		BeforeEach(func() {
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name: "noaction",
			}
		})

		It("should set SkipRemaining when current period is noaction", func() {
			nextCalled := false
			mockNext := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
			Expect(nextCalled).To(BeFalse())
		})
	})

	Context("When transitioning to noaction from an active period", func() {
		BeforeEach(func() {
			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name: "up",
			}
		})

		It("should continue to next handler to allow restore", func() {
			nextCalled := false
			mockNext := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Period).ToNot(BeNil())
			Expect(reconCtx.Period.Name).To(Equal("noaction"))
			Expect(reconCtx.SkipRemaining).To(BeFalse())
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("When run-once period was already processed", func() {
		BeforeEach(func() {
			now := time.Now()
			start := now.Add(-time.Minute).Format(time.DateTime)
			end := now.Add(time.Minute).Format(time.DateTime)

			scaler.Spec.Periods = []common.ScalerPeriod{
				{
					Name: "once-up",
					Type: "up",
					Time: common.TimePeriod{
						Fixed: &common.FixedPeriod{
							StartTime: start,
							EndTime:   end,
							Once:      ptr.To(true),
						},
					},
				},
			}

			curPeriod, err := periodPkg.New(&scaler.Spec.Periods[0])
			Expect(err).ToNot(HaveOccurred())
			Expect(curPeriod).ToNot(BeNil())

			scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name:    scaler.Spec.Periods[0].Name,
				Type:    scaler.Spec.Periods[0].Type,
				SpecSHA: curPeriod.Hash,
			}
		})

		It("should stop the chain and set requeue after period end", func() {
			nextCalled := false
			mockNext := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(BeNumerically(">", 0))
			Expect(nextCalled).To(BeFalse())
		})

		It("should cap requeue close to period remaining duration", func() {
			_ = handler.Execute(reconCtx)
			Expect(reconCtx.RequeueAfter).To(BeNumerically("<", 2*time.Minute+handlers.RequeueDelaySeconds*time.Second))
			Expect(reconCtx.RequeueAfter).To(BeNumerically(">", time.Second))
		})
	})

	It("should return critical error when period parsing fails", func() {
		scaler.Spec.Periods = []common.ScalerPeriod{
			{
				Name: "bad-period",
				Type: "up",
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						Days:      []string{"not-a-day"},
						StartTime: "09:00",
						EndTime:   "17:00",
						Once:      ptr.To(false),
					},
				},
			},
		}

		err := handler.Execute(reconCtx)
		Expect(err).To(HaveOccurred())
		Expect(service.IsCriticalError(err)).To(BeTrue(), fmt.Sprintf("expected critical error, got %v", err))
	})
})
