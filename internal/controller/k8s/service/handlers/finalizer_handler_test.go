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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/testutil"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

var _ = Describe("FinalizerHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		scaler   *kubecloudscalerv1alpha3.K8s
	)

	BeforeEach(func() {
		handler = handlers.NewFinalizerHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
		}

		reconCtx = &service.ReconciliationContext{
			Ctx: context.Background(),
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-scaler",
					Namespace: "default",
				},
			},
			Logger: &logger,
			Scaler: scaler,
		}
	})

	Context("When the scaler is not being deleted", func() {
		It("should add the finalizer if not present and continue", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

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
			Expect(controllerutil.ContainsFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)).To(BeTrue())
			Expect(nextCalled).To(BeTrue())
		})

	})

	Context("When the client Patch fails while adding finalizer", func() {
		It("should return a recoverable error and set RequeueAfter", func() {
			injectedErr := fmt.Errorf("persistent patch failure")
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return injectedErr
					},
				}).
				Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileErrorDuration))
		})
	})

	Context("When adding a finalizer succeeds", func() {
		It("should persist the finalizer via Patch and update ctx.Scaler", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(controllerutil.ContainsFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)).To(BeTrue())

			// Verify persisted on the server (not just in-memory)
			persisted := &kubecloudscalerv1alpha3.K8s{}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(persisted, handlers.ScalerFinalizer)).To(BeTrue())
		})
	})

	Context("When the scaler is being deleted with finalizer", func() {
		BeforeEach(func() {
			now := metav1.Now()
			scaler.ObjectMeta.DeletionTimestamp = &now
			controllerutil.AddFinalizer(scaler, handlers.ScalerFinalizer)
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()
		})

		It("should set ShouldFinalize flag and continue", func() {
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
			Expect(reconCtx.ShouldFinalize).To(BeTrue())
			Expect(nextCalled).To(BeTrue())
		})
	})

	Context("When the scaler is being deleted without finalizer", func() {
		It("should set SkipRemaining and not call next handler", func() {
			// Create a separate scaler without finalizer but with DeletionTimestamp
			// We simulate this by directly setting up the context scaler
			// (The fake client refuses to create objects with DeletionTimestamp but no finalizers)
			now := metav1.Now()
			scalerNoFinalizer := &kubecloudscalerv1alpha3.K8s{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-scaler-no-finalizer",
					Namespace:         "default",
					DeletionTimestamp: &now,
					Finalizers:        []string{}, // Empty finalizers
				},
			}
			// Use a client without the scaler since we won't be fetching it
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()
			reconCtx.Scaler = scalerNoFinalizer

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
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(nextCalled).To(BeFalse())
		})
	})
})
