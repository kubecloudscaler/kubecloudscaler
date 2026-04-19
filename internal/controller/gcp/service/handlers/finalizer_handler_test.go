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
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
)

var _ = Describe("FinalizerHandler", func() {
	var (
		logger           zerolog.Logger
		scheme           *runtime.Scheme
		finalizerHandler service.Handler
		reconCtx         *service.ReconciliationContext
		scaler           *kubecloudscalerv1alpha3.Gcp
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")

		finalizerHandler = handlers.NewFinalizerHandler()
	})

	Context("When scaler is not being deleted and has no finalizer", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()
			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: scaler.Name, Namespace: scaler.Namespace}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should add finalizer and continue", func() {
			err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.ShouldFinalize).To(BeFalse())
		})
	})

	Context("When client Patch fails while adding finalizer", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(
						ctx context.Context,
						c client.WithWatch,
						obj client.Object,
						patch client.Patch,
						opts ...client.PatchOption,
					) error {
						return fmt.Errorf("persistent patch failure")
					},
				}).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: scaler.Name, Namespace: scaler.Namespace}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should return a recoverable error and set RequeueAfter", func() {
			err := finalizerHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(5 * time.Second))
		})
	})

	Context("When adding the finalizer succeeds", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()
			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: scaler.Name, Namespace: scaler.Namespace}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should persist the finalizer via Patch and update ctx.Scaler", func() {
			Expect(finalizerHandler.Execute(reconCtx)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)).To(BeTrue())

			// Verify persisted on the server (not just in-memory).
			persisted := &kubecloudscalerv1alpha3.Gcp{}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(persisted, handlers.ScalerFinalizer)).To(BeTrue())
		})
	})

	Context("When scaler is being deleted with finalizer", func() {
		BeforeEach(func() {
			now := metav1.Now()
			scaler.SetDeletionTimestamp(&now)
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should set ShouldFinalize flag and continue", func() {
			err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.ShouldFinalize).To(BeTrue())
		})
	})

	Context("When scaler is being deleted without finalizer", func() {
		BeforeEach(func() {
			// Create without finalizer first
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			// Manually set deletion timestamp (simulating the scenario)
			now := metav1.Now()
			scaler.SetDeletionTimestamp(&now)

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should set SkipRemaining and stop chain", func() {
			err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})
	})

	Context("When the scaler is deleted between Fetch and finalizer Patch", func() {
		It("short-circuits without error and without requeue", func() {
			// Empty fake client — any Get inside patchAddFinalizer returns NotFound.
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: scaler.Name, Namespace: scaler.Namespace}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}

			err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.SkipRemaining).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(BeZero())
		})
	})

	Context("Retry-on-conflict when adding the finalizer", func() {
		It("retries once on a 409 and succeeds on the second Patch", func() {
			gvr := schema.GroupResource{Group: "kubecloudscaler.cloud", Resource: "gcps"}
			var patchCalls int
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
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

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: scaler.Name, Namespace: scaler.Namespace}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}

			Expect(finalizerHandler.Execute(reconCtx)).To(Succeed())
			Expect(patchCalls).To(Equal(2))
			persisted := &kubecloudscalerv1alpha3.Gcp{}
			Expect(k8sClient.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(persisted, handlers.ScalerFinalizer)).To(BeTrue())
		})
	})
})
