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

package k8s

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
		scaler := &kubecloudscalerv1alpha3.K8s{
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				Periods: []common.ScalerPeriod{
					{
						Type: "down",
						Time: common.TimePeriod{
							Recurring: &common.RecurringPeriod{
								Days: []string{
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

		BeforeEach(func() {
			By("creating the custom resource for the Kind Scaler")
			err := testEnv.Client.Get(ctx, typeNamespacedName, scaler)
			if err != nil && errors.IsNotFound(err) {
				resource := &kubecloudscalerv1alpha3.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kubecloudscalerv1alpha3.K8sSpec{
						Periods: []common.ScalerPeriod{
							{
								Type: "down",
								Time: common.TimePeriod{
									Recurring: &common.RecurringPeriod{
										Days: []string{
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
				Expect(testEnv.Client.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "down",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days: []string{
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
			err := testEnv.Client.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Scaler")
			Expect(testEnv.Client.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := NewScalerReconciler(
				testEnv.Client,
				testEnv.Client.Scheme(),
				&log.Logger,
			)

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle non-existent resource gracefully", func() {
			By("Reconciling a non-existent resource")
			controllerReconciler := NewScalerReconciler(
				testEnv.Client,
				testEnv.Client.Scheme(),
				&log.Logger,
			)

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
