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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/testutil"
)

var _ = Describe("FetchHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
	)

	BeforeEach(func() {
		handler = handlers.NewFetchHandler()
		logger = zerolog.Nop() // Use a no-op logger for tests
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		reconCtx = &service.ReconciliationContext{
			Ctx: context.Background(),
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-scaler",
					Namespace: "default",
				},
			},
			Logger: &logger,
		}
	})

	Context("When the Scaler resource exists", func() {
		It("should fetch the scaler and add it to the context", func() {
			scaler := &kubecloudscalerv1alpha3.K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-scaler",
					Namespace: "default",
				},
			}
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler).ToNot(BeNil())
			Expect(reconCtx.Scaler.Name).To(Equal("test-scaler"))
		})

	})

	Context("When the Scaler resource does not exist", func() {
		It("should return nil so controller-runtime does not log a spurious Reconciler error", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build() // Client without the scaler

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler).To(BeNil())
		})
	})

	Context("When Get returns a non-NotFound error", func() {
		It("should return a recoverable error", func() {
			injectedErr := fmt.Errorf("transient API failure")
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						return injectedErr
					},
				}).
				Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.Scaler).To(BeNil())
		})
	})

	Context("When setNext is called", func() {
		It("should chain to the next handler on success", func() {
			scaler := &kubecloudscalerv1alpha3.K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-scaler",
					Namespace: "default",
				},
			}
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			// Create a mock next handler that tracks if it was called
			nextCalled := false
			mockNext := &testutil.MockHandler{
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

		It("should not chain to the next handler when resource is not found", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build() // Client without the scaler

			nextCalled := false
			mockNext := &testutil.MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			}
			handler.SetNext(mockNext)

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(nextCalled).To(BeFalse())
		})
	})
})
