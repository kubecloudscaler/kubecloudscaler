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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
)

const flowFinalizerConst = "kubecloudscaler.cloud/flow-finalizer"

var _ = Describe("FinalizerHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.FlowReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		flow     *kubecloudscalerv1alpha3.Flow
	)

	BeforeEach(func() {
		handler = handlers.NewFinalizerHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		flow = &kubecloudscalerv1alpha3.Flow{
			ObjectMeta: metav1.ObjectMeta{Name: "test-flow"},
		}
		reconCtx = &service.FlowReconciliationContext{
			Ctx: context.Background(),
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{Name: flow.Name},
			},
			Logger: &logger,
			Flow:   flow,
		}
	})

	Context("When the flow is not being deleted and has no finalizer", func() {
		It("persists the finalizer via Patch and updates ctx.Flow", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(flow).Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(reconCtx.Flow, flowFinalizerConst)).To(BeTrue())

			persisted := &kubecloudscalerv1alpha3.Flow{}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(persisted, flowFinalizerConst)).To(BeTrue())
		})
	})

	Context("When Patch fails while adding the finalizer", func() {
		It("returns a RecoverableError and sets RequeueAfter", func() {
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(flow).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return fmt.Errorf("persistent patch failure")
					},
				}).
				Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(shared.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(BeNumerically(">", 0))
		})
	})

	Context("When the flow is being deleted with a finalizer", func() {
		It("removes the finalizer via Patch and stops the chain", func() {
			controllerutil.AddFinalizer(flow, flowFinalizerConst)
			now := metav1.Now()
			flow.SetDeletionTimestamp(&now)
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(flow).Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(controllerutil.ContainsFinalizer(reconCtx.Flow, flowFinalizerConst)).To(BeFalse())
		})
	})

	Context("When the flow is being deleted without a finalizer", func() {
		It("short-circuits the chain without error", func() {
			now := metav1.Now()
			flow.SetDeletionTimestamp(&now)
			// Note: fake client refuses to create objects with DeletionTimestamp and no finalizers.
			// Use an empty store; the finalizer handler only reads ctx.Flow in-memory here.
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})
	})

	Context("Retry-on-conflict when adding the finalizer", func() {
		It("retries once on a 409 and succeeds on the second Patch", func() {
			gvr := schema.GroupResource{Group: "kubecloudscaler.cloud", Resource: "flows"}
			var patchCalls int
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(flow).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						patchCalls++
						if patchCalls == 1 {
							return apierrors.NewConflict(gvr, obj.GetName(), fmt.Errorf("conflict"))
						}
						return c.Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(patchCalls).To(Equal(2))
			Expect(controllerutil.ContainsFinalizer(reconCtx.Flow, flowFinalizerConst)).To(BeTrue())
		})
	})

	Context("Retry-on-conflict when removing the finalizer", func() {
		It("retries once on a 409 and succeeds on the second Patch", func() {
			controllerutil.AddFinalizer(flow, flowFinalizerConst)
			now := metav1.Now()
			flow.SetDeletionTimestamp(&now)
			gvr := schema.GroupResource{Group: "kubecloudscaler.cloud", Resource: "flows"}
			var patchCalls int
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(flow).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						patchCalls++
						if patchCalls == 1 {
							return apierrors.NewConflict(gvr, obj.GetName(), fmt.Errorf("conflict"))
						}
						return c.Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(patchCalls).To(Equal(2))
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})
	})

	Context("When Patch fails while removing the finalizer", func() {
		It("returns a RecoverableError and sets RequeueAfter", func() {
			controllerutil.AddFinalizer(flow, flowFinalizerConst)
			now := metav1.Now()
			flow.SetDeletionTimestamp(&now)
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(flow).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
						return fmt.Errorf("persistent patch failure")
					},
				}).
				Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(shared.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(BeNumerically(">", 0))
			Expect(reconCtx.SkipRemaining).To(BeFalse())
		})
	})

	Context("When Get returns NotFound while adding the finalizer", func() {
		It("short-circuits without error (flow was deleted between Fetch and Patch)", func() {
			// Fake client with no flow stored → Get returns NotFound.
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(BeZero())
		})
	})

	Context("When Get returns NotFound while removing the finalizer", func() {
		It("short-circuits without error (cleanup already complete)", func() {
			controllerutil.AddFinalizer(flow, flowFinalizerConst)
			now := metav1.Now()
			flow.SetDeletionTimestamp(&now)
			// Object on ctx.Flow but absent from the API server — simulates already-deleted state.
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})
	})
})
