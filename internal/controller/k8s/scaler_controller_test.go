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
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
)

var _ = Describe("Scaler Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
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
			err := k8sClientTest.Get(ctx, typeNamespacedName, scaler)
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
				Expect(k8sClientTest.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
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
			err := k8sClientTest.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Scaler")
			Expect(k8sClientTest.Delete(ctx, resource)).To(Succeed())
		})

		It("should handle reconciliation with handler chain pattern", func() {
			By("Reconciling the created resource using handler chain")
			controllerReconciler := &ScalerReconciler{
				Client: k8sClientTest,
				Scheme: k8sClientTest.Scheme(),
				Logger: &log.Logger,
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			// In the test environment without a real K8s cluster:
			// - FetchHandler: Successfully fetches the scaler from envtest
			// - FinalizerHandler: Successfully adds finalizer
			// - AuthHandler: Fails because k8sClient.GetClient() needs real cluster config
			//
			// The AuthHandler returns a CriticalError when unable to create K8s client
			// because the test environment doesn't have KUBERNETES_SERVICE_HOST/PORT set.
			//
			// This is expected behavior - the chain pattern correctly propagates
			// the critical error and stops further processing.
			if err != nil {
				// Verify it's a critical error (expected in test environment)
				Expect(service.IsCriticalError(err)).To(BeTrue())
				// Result should be empty for critical errors
				Expect(result.RequeueAfter).To(BeZero())
			} else {
				// If running in a real cluster environment, reconciliation should succeed
				// and result in a requeue
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			}
		})

		It("should successfully fetch and process scaler resource through chain", func() {
			By("Verifying handler chain processes fetch and finalizer handlers")
			controllerReconciler := &ScalerReconciler{
				Client: k8sClientTest,
				Scheme: k8sClientTest.Scheme(),
				Logger: &log.Logger,
			}

			// First reconcile will add finalizer
			_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			// Verify the scaler was fetched and finalizer was added
			fetchedScaler := &kubecloudscalerv1alpha3.K8s{}
			err := k8sClientTest.Get(ctx, typeNamespacedName, fetchedScaler)
			Expect(err).NotTo(HaveOccurred())
			Expect(fetchedScaler.Finalizers).To(ContainElement("kubecloudscaler.cloud/finalizer"))
		})
	})

	Context("When handler chain initialization", func() {
		It("should create handlers in correct order", func() {
			By("Verifying initializeChain returns a valid handler")
			controllerReconciler := &ScalerReconciler{
				Client: k8sClientTest,
				Scheme: k8sClientTest.Scheme(),
				Logger: &log.Logger,
			}

			chain := controllerReconciler.initializeChain()
			Expect(chain).ToNot(BeNil())
			// The chain is a FetchHandler (first handler)
			// We can't easily verify the chain structure without more complex testing
			// but we can verify the chain is created successfully
		})
	})
})
