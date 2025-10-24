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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("Flow Controller Integration Tests", func() {
	var (
		ctx        context.Context
		reconciler *FlowReconciler
		scheme     *runtime.Scheme
		fakeClient client.Client
		logger     *zerolog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = &log.Logger
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&kubecloudscalerv1alpha3.Flow{}).
			WithStatusSubresource(&kubecloudscalerv1alpha3.K8s{}).
			WithStatusSubresource(&kubecloudscalerv1alpha3.Gcp{}).
			Build()

		reconciler = &FlowReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Logger: logger,
		}
	})

	Context("When processing flows with mocked dependencies", func() {
		It("should handle successful flow processing", func() {
			By("Creating a flow resource")
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-flow-mock",
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
								{Name: "test-k8s"},
							},
						},
					},
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			By("Reconciling the flow")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      flow.Name,
					Namespace: flow.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("Verifying the flow status was updated")
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(flow), updatedFlow)).To(Succeed())
			Expect(updatedFlow.Status.Conditions).To(HaveLen(1))
			Expect(updatedFlow.Status.Conditions[0].Type).To(Equal("Processed"))
			Expect(updatedFlow.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))

			By("Verifying the K8s resource was created")
			k8sResource := &kubecloudscalerv1alpha3.K8s{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      "flow-test-flow-mock-test-k8s",
				Namespace: "default",
			}, k8sResource)).To(Succeed())
			Expect(k8sResource.Spec.Periods).To(HaveLen(1))
		})

		It("should handle flow processing errors gracefully", func() {
			By("Creating a flow with invalid configuration")
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow-error",
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
							PeriodName: "invalid-period", // This will cause an error
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-k8s"},
							},
						},
					},
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			By("Reconciling the flow should fail")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      flow.Name,
					Namespace: flow.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred()) // Reconcile doesn't return error, it updates status
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("Verifying the flow status shows error")
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(flow), updatedFlow)).To(Succeed())
			Expect(updatedFlow.Status.Conditions).To(HaveLen(1))
			Expect(updatedFlow.Status.Conditions[0].Type).To(Equal("Processed"))
			Expect(updatedFlow.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(updatedFlow.Status.Conditions[0].Reason).To(Equal("ProcessingFailed"))
		})

		It("should handle flow deletion with finalizer", func() {
			By("Creating a flow with finalizer")
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-flow-delete",
					Namespace:  "default",
					Finalizers: []string{"kubecloudscaler.cloud/flow-finalizer"},
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
								{Name: "test-k8s"},
							},
						},
					},
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			By("Setting deletion timestamp")
			now := metav1.Now()
			flow.DeletionTimestamp = &now
			Expect(fakeClient.Update(ctx, flow)).To(Succeed())

			By("Reconciling should remove finalizer")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      flow.Name,
					Namespace: flow.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Verifying finalizer was removed")
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(flow), updatedFlow)).To(Succeed())
			Expect(updatedFlow.Finalizers).NotTo(ContainElement("kubecloudscaler.cloud/flow-finalizer"))
		})

		It("should handle GCP resources correctly", func() {
			By("Creating a flow with GCP resource")
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow-gcp",
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
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			By("Reconciling the flow")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      flow.Name,
					Namespace: flow.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("Verifying the GCP resource was created")
			gcpResource := &kubecloudscalerv1alpha3.Gcp{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      "flow-test-flow-gcp-test-gcp",
				Namespace: "default",
			}, gcpResource)).To(Succeed())
			Expect(gcpResource.Spec.Periods).To(HaveLen(1))
		})

		It("should handle flows with timing delays", func() {
			By("Creating a flow with timing delays")
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow-delays",
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
									StartTimeDelay: "30m",
									EndTimeDelay:   "15m",
								},
							},
						},
					},
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			By("Reconciling the flow")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      flow.Name,
					Namespace: flow.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("Verifying the K8s resource was created with adjusted timing")
			k8sResource := &kubecloudscalerv1alpha3.K8s{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      "flow-test-flow-delays-test-k8s",
				Namespace: "default",
			}, k8sResource)).To(Succeed())
			Expect(k8sResource.Spec.Periods).To(HaveLen(1))
			// The start time should be adjusted by the delay
			Expect(k8sResource.Spec.Periods[0].Time.Recurring.StartTime).To(Equal("09:30"))
		})
	})

	Context("When handling edge cases", func() {
		It("should handle flow not found gracefully", func() {
			By("Reconciling a non-existent flow")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-flow",
					Namespace: "default",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
		})

		It("should handle flows with empty periods", func() {
			By("Creating a flow with empty periods")
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow-empty-periods",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{},
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
							PeriodName: "non-existent-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-k8s"},
							},
						},
					},
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			By("Reconciling should fail with period not found error")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      flow.Name,
					Namespace: flow.Namespace,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("Verifying the flow status shows error")
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(flow), updatedFlow)).To(Succeed())
			Expect(updatedFlow.Status.Conditions).To(HaveLen(1))
			Expect(updatedFlow.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
		})
	})
})
