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

package service

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("StatusUpdaterService", func() {
	var (
		scheme    *runtime.Scheme
		logger    zerolog.Logger
		svc       *StatusUpdaterService
		k8sClient client.Client
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		logger = zerolog.Nop()
		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&kubecloudscalerv1alpha3.Flow{}).
			Build()
		svc = NewStatusUpdaterService(k8sClient, &logger)
	})

	Describe("UpdateFlowStatus", func() {
		Context("when updating with a success condition", func() {
			It("should update the flow status successfully", func() {
				flow := &kubecloudscalerv1alpha3.Flow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-flow",
						Namespace: "default",
					},
					Status: kubecloudscalerv1alpha3.FlowStatus{
						Conditions: []metav1.Condition{},
					},
				}

				err := k8sClient.Create(context.Background(), flow)
				Expect(err).ToNot(HaveOccurred())

				condition := metav1.Condition{
					Type:    "Processed",
					Status:  metav1.ConditionTrue,
					Reason:  "ProcessingSucceeded",
					Message: "Flow processed successfully",
				}

				err = svc.UpdateFlowStatus(context.Background(), flow, condition)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when updating with an error condition", func() {
			It("should update the flow status successfully", func() {
				flow := &kubecloudscalerv1alpha3.Flow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-flow-error",
						Namespace: "default",
					},
					Status: kubecloudscalerv1alpha3.FlowStatus{
						Conditions: []metav1.Condition{},
					},
				}

				err := k8sClient.Create(context.Background(), flow)
				Expect(err).ToNot(HaveOccurred())

				condition := metav1.Condition{
					Type:    "Error",
					Status:  metav1.ConditionTrue,
					Reason:  "ProcessingFailed",
					Message: "Flow processing failed",
				}

				err = svc.UpdateFlowStatus(context.Background(), flow, condition)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
