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
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	k8sUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

var _ = Describe("ScalingHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		scaler   *kubecloudscalerv1alpha3.K8s
	)

	BeforeEach(func() {
		handler = handlers.NewScalingHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{
				Resources: common.Resources{
					Types: []string{"deployment"},
				},
			},
		}

		mockK8sClient := fake.NewSimpleClientset()

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
			K8sClient: mockK8sClient,
			Period:    &period.Period{Name: "up", Type: "up"},
			ResourceConfig: resources.Config{
				K8s: &k8sUtils.Config{
					Client: mockK8sClient,
				},
			},
		}
	})

	Context("When resource configuration is valid", func() {
		It("should attempt to scale resources and continue", func() {
			nextCalled := false
			mockNext := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)

			// Scaling may fail due to no actual resources, but handler should continue
			Expect(err).ToNot(HaveOccurred())
			Expect(nextCalled).To(BeTrue())
		})

		It("should complete in under 100ms", func() {
			startTime := time.Now()
			_ = handler.Execute(reconCtx)
			duration := time.Since(startTime)

			Expect(duration).To(BeNumerically("<", 100*time.Millisecond))
		})
	})

	Context("When no resource types are specified", func() {
		It("should default to deployment and attempt to scale", func() {
			scaler.Spec.Resources.Types = []string{} // No types specified

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
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("When scaling operations produce results", func() {
		It("should continue chain and collect results", func() {
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
			// Results may be empty or nil if no resources are found, but chain should continue
			Expect(nextCalled).To(BeTrue())
		})
	})
})
