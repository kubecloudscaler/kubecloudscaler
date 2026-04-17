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
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
)

var _ = Describe("FetchHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.FlowReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
	)

	BeforeEach(func() {
		handler = handlers.NewFetchHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		reconCtx = &service.FlowReconciliationContext{
			Ctx: context.Background(),
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "test-flow"},
			},
			Logger: &logger,
		}
	})

	Context("When the Flow resource exists", func() {
		It("populates ctx.Flow and continues the chain", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow"},
			}
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(flow).Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.Flow).ToNot(BeNil())
			Expect(reconCtx.Flow.Name).To(Equal("test-flow"))
		})
	})

	Context("When the Flow resource is not found", func() {
		It("returns a CriticalError so the controller ignores NotFound without requeue", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(shared.IsCriticalError(err)).To(BeTrue())
			Expect(reconCtx.Flow).To(BeNil())
		})
	})

	Context("When the API Get returns a non-NotFound error", func() {
		It("returns a RecoverableError to trigger requeue", func() {
			injected := fmt.Errorf("transient API failure")
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						return injected
					},
				}).
				Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(shared.IsRecoverableError(err)).To(BeTrue())
		})
	})
})
