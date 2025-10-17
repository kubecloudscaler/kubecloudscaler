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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
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
		logger = &zerolog.Logger{}
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
					Periods: []*common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("bigup"),
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
									Name:  "test-k8s-resource",
									Delay: ptr.To("600s"),
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
		})

	})
})
