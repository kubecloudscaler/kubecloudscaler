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

package gcp

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("Scaler Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		scaler := &kubecloudscalerv1alpha3.Gcp{
			Spec: kubecloudscalerv1alpha3.GcpSpec{
				Periods: []common.ScalerPeriod{
					{
						Type: "down",
						Time: common.TimePeriod{
							Recurring: &common.RecurringPeriod{
								Days: []common.DayOfWeek{
									common.DayAll,
								},
								StartTime: "00:00",
								EndTime:   "00:00",
								Once:      ptr.To(false),
							},
						},
					},
				},
			},
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Scaler")
			err := k8sClient.Get(ctx, typeNamespacedName, scaler)
			if err != nil && errors.IsNotFound(err) {
				resource := &kubecloudscalerv1alpha3.Gcp{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kubecloudscalerv1alpha3.GcpSpec{
						Config: kubecloudscalerv1alpha3.GcpConfig{
							DefaultPeriodType: "down",
						},
						Periods: []common.ScalerPeriod{
							{
								Type: "down",
								Time: common.TimePeriod{
									Recurring: &common.RecurringPeriod{
										Days: []common.DayOfWeek{
											"all",
										},
										StartTime: "00:00",
										EndTime:   "00:00",
										Once:      ptr.To(false),
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						DefaultPeriodType: "down",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: "down",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days: []common.DayOfWeek{
										common.DayAll,
									},
									StartTime: "00:00",
									EndTime:   "00:00",
									Once:      ptr.To(false),
								},
							},
						},
					},
				},
			}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil && errors.IsNotFound(err) {
				// Resource was deleted by the test itself; nothing to clean up.
				return
			}
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Scaler")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should gracefully handle a NotFound via the handler chain", func() {
			By("Deleting the resource before reconciling")
			existing := &kubecloudscalerv1alpha3.Gcp{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, existing)).To(Succeed())
			Expect(k8sClient.Delete(ctx, existing)).To(Succeed())

			By("Reconciling the deleted resource")
			controllerReconciler := &ScalerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Logger: &log.Logger,
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			// FetchHandler must swallow NotFound without requeue or error, so controller-runtime
			// does not log a spurious "Reconciler error".
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
		})
	})
})
