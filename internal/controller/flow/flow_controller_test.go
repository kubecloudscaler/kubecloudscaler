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

package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("Flow Controller", func() {
	var (
		ctx                  context.Context
		controllerReconciler *FlowReconciler
		logger               *zerolog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = &log.Logger
		controllerReconciler = &FlowReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			Logger: logger,
		}
	})

	Context("When reconciling a Flow resource", func() {
		const resourceName = "test-flow"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		var flow *kubecloudscalerv1alpha3.Flow

		BeforeEach(func() {
			By("creating the custom resource for the Kind Flow")
			flow = &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: "bigup",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "test-k8s-resource",
								Resources: common.Resources{
									Types: []string{"deployments"},
									Names: []string{"test-deployment"},
								},
								Config: kubecloudscalerv1alpha3.K8sConfig{
									Namespaces: []string{"default"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "bigup",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:           "test-k8s-resource",
									StartTimeDelay: "10m",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, flow)).To(Succeed())
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance Flow")
			resource := &kubecloudscalerv1alpha3.Flow{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource and create K8s objects", func() {
			By("Reconciling the created resource")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("Verifying the Flow status is updated")
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedFlow)).To(Succeed())
			Expect(updatedFlow.Status.Conditions).To(HaveLen(1))
			Expect(updatedFlow.Status.Conditions[0].Type).To(Equal("Processed"))
			Expect(updatedFlow.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))

			By("Verifying K8s resource was created with correct periods")
			k8sResource := &kubecloudscalerv1alpha3.K8s{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "flow-test-flow-test-k8s-resource",
				Namespace: "default",
			}, k8sResource)).To(Succeed())
			Expect(k8sResource.Spec.Periods).To(HaveLen(1))
			Expect(k8sResource.Spec.Periods[0].Name).To(Equal("bigup"))
		})

		Context("When handling different flow configurations", func() {
			DescribeTable("should handle various flow configurations correctly",
				func(flowSpec kubecloudscalerv1alpha3.FlowSpec, expectedError bool, description string) {
					By(fmt.Sprintf("Testing: %s", description))

					testFlow := &kubecloudscalerv1alpha3.Flow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("test-flow-%d", time.Now().UnixNano()),
							Namespace: "default",
						},
						Spec: flowSpec,
					}

					Expect(k8sClient.Create(ctx, testFlow)).To(Succeed())
					defer func() {
						Expect(k8sClient.Delete(ctx, testFlow)).To(Succeed())
					}()

					result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      testFlow.Name,
							Namespace: testFlow.Namespace,
						},
					})

					if expectedError {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
						Expect(result.RequeueAfter).To(BeNumerically(">", 0))
					}
				},
				Entry("Valid flow with K8s resource", kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "test-k8s",
								Resources: common.Resources{
									Types: []string{"deployments"},
									Names: []string{"test-deployment"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-k8s"},
							},
						},
					},
				}, false, "Valid flow with K8s resource"),

				Entry("Valid flow with GCP resource", kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						Gcp: []kubecloudscalerv1alpha3.GcpResource{
							{
								Name: "test-gcp",
								Resources: common.Resources{
									Types: []string{"vm-instances"},
									Names: []string{"test-instance"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-gcp"},
							},
						},
					},
				}, false, "Valid flow with GCP resource"),

				Entry("Flow with invalid period reference", kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("valid-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "test-k8s",
								Resources: common.Resources{
									Types: []string{"deployments"},
									Names: []string{"test-deployment"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "invalid-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-k8s"},
							},
						},
					},
				}, true, "Flow with invalid period reference"),

				Entry("Flow with invalid resource reference", kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "valid-k8s",
								Resources: common.Resources{
									Types: []string{"deployments"},
									Names: []string{"test-deployment"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "invalid-resource"},
							},
						},
					},
				}, true, "Flow with invalid resource reference"),
			)
		})

		Context("When handling timing delays", func() {
			It("should correctly calculate start and end times with delays", func() {
				By("Creating a flow with timing delays")
				flowWithDelays := &kubecloudscalerv1alpha3.Flow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-flow-delays",
						Namespace: "default",
					},
					Spec: kubecloudscalerv1alpha3.FlowSpec{
						Periods: []common.ScalerPeriod{
							{
								Type: "up",
								Name: ptr.To("delayed-period"),
								Time: common.TimePeriod{
									Recurring: &common.RecurringPeriod{
										Days:      []string{"Monday"},
										StartTime: "09:00",
										EndTime:   "17:00",
									},
								},
							},
						},
						Resources: kubecloudscalerv1alpha3.Resources{
							K8s: []kubecloudscalerv1alpha3.K8sResource{
								{
									Name: "delayed-k8s",
									Resources: common.Resources{
										Types: []string{"deployments"},
										Names: []string{"test-deployment"},
									},
								},
							},
						},
						Flows: []kubecloudscalerv1alpha3.Flows{
							{
								PeriodName: "delayed-period",
								Resources: []kubecloudscalerv1alpha3.FlowResource{
									{
										Name:           "delayed-k8s",
										StartTimeDelay: "30m",
										EndTimeDelay:   "15m",
									},
								},
							},
						},
					},
				}

				Expect(k8sClient.Create(ctx, flowWithDelays)).To(Succeed())
				defer func() {
					Expect(k8sClient.Delete(ctx, flowWithDelays)).To(Succeed())
				}()

				By("Reconciling the flow with delays")
				result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      flowWithDelays.Name,
						Namespace: flowWithDelays.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				By("Verifying the K8s resource was created with adjusted timing")
				k8sResource := &kubecloudscalerv1alpha3.K8s{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flow-test-flow-delays-delayed-k8s",
					Namespace: "default",
				}, k8sResource)).To(Succeed())

				Expect(k8sResource.Spec.Periods).To(HaveLen(1))
				// The start time should be adjusted by the delay
				Expect(k8sResource.Spec.Periods[0].Time.Recurring.StartTime).To(Equal("09:30"))
			})
		})

		Context("When handling deletion", func() {
			It("should properly handle finalizer cleanup", func() {
				By("Reconciling the flow to add finalizer")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying finalizer was added")
				updatedFlow := &kubecloudscalerv1alpha3.Flow{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, updatedFlow)).To(Succeed())
				Expect(updatedFlow.Finalizers).To(ContainElement("kubecloudscaler.cloud/flow-finalizer"))

				By("Deleting the flow")
				Expect(k8sClient.Delete(ctx, updatedFlow)).To(Succeed())

				By("Reconciling after deletion")
				result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
			})
		})

		Context("When handling errors", func() {
			It("should handle invalid delay formats gracefully", func() {
				By("Creating a flow with invalid delay format")
				invalidFlow := &kubecloudscalerv1alpha3.Flow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-flow-invalid-delay",
						Namespace: "default",
					},
					Spec: kubecloudscalerv1alpha3.FlowSpec{
						Periods: []common.ScalerPeriod{
							{
								Type: "up",
								Name: ptr.To("test-period"),
								Time: common.TimePeriod{
									Recurring: &common.RecurringPeriod{
										Days:      []string{"Monday"},
										StartTime: "09:00",
										EndTime:   "17:00",
									},
								},
							},
						},
						Resources: kubecloudscalerv1alpha3.Resources{
							K8s: []kubecloudscalerv1alpha3.K8sResource{
								{
									Name: "test-k8s",
									Resources: common.Resources{
										Types: []string{"deployments"},
										Names: []string{"test-deployment"},
									},
								},
							},
						},
						Flows: []kubecloudscalerv1alpha3.Flows{
							{
								PeriodName: "test-period",
								Resources: []kubecloudscalerv1alpha3.FlowResource{
									{
										Name:           "test-k8s",
										StartTimeDelay: "invalid-delay",
									},
								},
							},
						},
					},
				}

				Expect(k8sClient.Create(ctx, invalidFlow)).To(Succeed())
				defer func() {
					Expect(k8sClient.Delete(ctx, invalidFlow)).To(Succeed())
				}()

				By("Reconciling should fail with invalid delay")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      invalidFlow.Name,
						Namespace: invalidFlow.Namespace,
					},
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid start time delay format"))
			})
		})
	})
})
