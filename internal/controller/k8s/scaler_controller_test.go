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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
)

var _ = Describe("Scaler Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		scaler := &kubecloudscalerv1alpha1.K8s{
			Spec: kubecloudscalerv1alpha1.K8sSpec{
				Periods: []*kubecloudscalerv1alpha1.ScalerPeriod{
					{
						Type: "down",
						Time: kubecloudscalerv1alpha1.TimePeriod{
							Recurring: &kubecloudscalerv1alpha1.RecurringPeriod{
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
			err := k8sClientTest.Get(ctx, typeNamespacedName, scaler)
			if err != nil && errors.IsNotFound(err) {
				resource := &kubecloudscalerv1alpha1.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kubecloudscalerv1alpha1.K8sSpec{
						Periods: []*kubecloudscalerv1alpha1.ScalerPeriod{
							{
								Type: "down",
								Time: kubecloudscalerv1alpha1.TimePeriod{
									Recurring: &kubecloudscalerv1alpha1.RecurringPeriod{
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
				Expect(k8sClientTest.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kubecloudscalerv1alpha1.K8s{
				Spec: kubecloudscalerv1alpha1.K8sSpec{
					Periods: []*kubecloudscalerv1alpha1.ScalerPeriod{
						{
							Type: "down",
							Time: kubecloudscalerv1alpha1.TimePeriod{
								Recurring: &kubecloudscalerv1alpha1.RecurringPeriod{
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
			err := k8sClientTest.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Scaler")
			Expect(k8sClientTest.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ScalerReconciler{
				Client: k8sClientTest,
				Scheme: k8sClientTest.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
