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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
)

var _ = Describe("FetchHandler", func() {
	var (
		logger       zerolog.Logger
		scheme       *runtime.Scheme
		k8sClient    client.Client
		fetchHandler service.Handler
		reconCtx     *service.ReconciliationContext
		scalerName   string
		scalerNS     string
	)

	BeforeEach(func() {
		logger = zerolog.Nop() // No-op logger for tests
		scalerName = "test-scaler"
		scalerNS = "default"

		// Setup scheme
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		// Create reconciliation context
		reconCtx = &service.ReconciliationContext{
			Ctx: context.Background(),
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      scalerName,
					Namespace: scalerNS,
				},
			},
			Logger: &logger,
		}
	})

	Context("When scaler resource exists", func() {
		BeforeEach(func() {
			// Create a scaler resource
			scaler := &kubecloudscalerv1alpha3.Gcp{}
			scaler.SetName(scalerName)
			scaler.SetNamespace(scalerNS)

			// Create fake client with the scaler resource
			k8sClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx.Client = k8sClient

			// Create handler
			fetchHandler = handlers.NewFetchHandler()
		})

		It("should fetch the scaler resource and populate context", func() {
			err := fetchHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler).ToNot(BeNil())
			Expect(reconCtx.Scaler.Name).To(Equal(scalerName))
			Expect(reconCtx.Scaler.Namespace).To(Equal(scalerNS))
		})

	})

	Context("When scaler resource does not exist", func() {
		BeforeEach(func() {
			// Create fake client without the scaler resource
			k8sClient = fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconCtx.Client = k8sClient

			// Create handler
			fetchHandler = handlers.NewFetchHandler()
		})

		It("should return nil and set SkipRemaining (resource deleted gracefully)", func() {
			err := fetchHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(reconCtx.Scaler).To(BeNil())
		})
	})

	Context("When client returns a transient error", func() {
		BeforeEach(func() {
			k8sClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						ctx context.Context,
						c client.WithWatch,
						key client.ObjectKey,
						obj client.Object,
						opts ...client.GetOption,
					) error {
						return fmt.Errorf("transient API failure")
					},
				}).
				Build()

			reconCtx.Client = k8sClient
			fetchHandler = handlers.NewFetchHandler()
		})

		It("should return a recoverable error", func() {
			err := fetchHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.Scaler).To(BeNil())
			Expect(reconCtx.RequeueAfter).To(Equal(5 * time.Second))
		})
	})
})
