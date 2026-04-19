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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

var _ = Describe("StatusHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		scaler   *kubecloudscalerv1alpha3.K8s
	)

	BeforeEach(func() {
		handler = handlers.NewStatusHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
			Status: common.ScalerStatus{
				CurrentPeriod: &common.ScalerStatusPeriod{},
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
			Client:         fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).WithStatusSubresource(scaler).Build(),
			Logger:         &logger,
			Scaler:         scaler,
			SuccessResults: []common.ScalerStatusSuccess{},
			FailedResults:  []common.ScalerStatusFailed{},
		}
	})

	Context("When updating status with successful results", func() {
		It("should update status and set requeue", func() {
			reconCtx.SuccessResults = []common.ScalerStatusSuccess{
				{Kind: "deployment", Name: "test-deployment-1", Comment: "scaled up"},
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Successful).To(Equal(reconCtx.SuccessResults))
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})

	})

	Context("When updating status with failed results", func() {
		It("should update status with failures and set requeue", func() {
			reconCtx.FailedResults = []common.ScalerStatusFailed{
				{Kind: "deployment", Name: "test-deployment-2", Reason: "API error"},
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Failed).To(Equal(reconCtx.FailedResults))
		})
	})

	Context("When finalizer cleanup is requested", func() {
		It("should remove the finalizer and not set requeue", func() {
			controllerutil.AddFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(reconCtx.Scaler).Build()
			reconCtx.ShouldFinalize = true

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(controllerutil.ContainsFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)).To(BeFalse())
		})
	})

	Context("When client Patch fails during finalizer removal", func() {
		It("should return a recoverable error and set RequeueAfter", func() {
			controllerutil.AddFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)
			injectedErr := fmt.Errorf("persistent patch failure")
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(reconCtx.Scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return injectedErr
					},
				}).
				Build()
			reconCtx.ShouldFinalize = true

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileErrorDuration))
		})
	})

	Context("When client Status Patch fails", func() {
		It("should return a recoverable error and set RequeueAfter", func() {
			injectedErr := fmt.Errorf("status patch failure")
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(
						_ context.Context,
						_ client.Client,
						_ string,
						_ client.Object,
						_ client.Patch,
						_ ...client.SubResourcePatchOption,
					) error {
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

	Context("When both success and failed results exist", func() {
		It("should update status with both success and failures", func() {
			reconCtx.SuccessResults = []common.ScalerStatusSuccess{
				{Kind: "deployment", Name: "test-deployment-1", Comment: "scaled up"},
			}
			reconCtx.FailedResults = []common.ScalerStatusFailed{
				{Kind: "deployment", Name: "test-deployment-2", Reason: "API error"},
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Successful).To(Equal(reconCtx.SuccessResults))
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Failed).To(Equal(reconCtx.FailedResults))
		})
	})

	Context("When this is the last handler in chain", func() {
		It("should not call next handler when next is nil", func() {
			// Default handler has no next set
			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("DeepCopy isolation between snapshot and Get-overwritten latest", func() {
		It("writes the ctx-sourced Successful/Failed slices, not the server-side state", func() {
			// Pre-populate the server with different slice contents than what ctx holds. A
			// shallow copy of CurrentPeriod would alias the slices and let server state bleed
			// through after the retry loop's Get.
			serverScaler := scaler.DeepCopy()
			serverScaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Successful: []common.ScalerStatusSuccess{{Name: "stale-from-server", Kind: "deployment"}},
				Failed:     []common.ScalerStatusFailed{{Name: "stale-failed", Kind: "deployment", Reason: "old"}},
			}
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(serverScaler).
				WithStatusSubresource(serverScaler).
				Build()
			reconCtx.SuccessResults = []common.ScalerStatusSuccess{{Name: "fresh-success", Kind: "deployment"}}
			reconCtx.FailedResults = []common.ScalerStatusFailed{}

			Expect(handler.Execute(reconCtx)).To(Succeed())

			persisted := &kubecloudscalerv1alpha3.K8s{}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(persisted.Status.CurrentPeriod).ToNot(BeNil())
			Expect(persisted.Status.CurrentPeriod.Successful).To(ConsistOf(
				common.ScalerStatusSuccess{Name: "fresh-success", Kind: "deployment"},
			))
			Expect(persisted.Status.CurrentPeriod.Failed).To(BeEmpty())

			// Mutating the ctx slice after the write must not shift persisted state (no alias).
			reconCtx.SuccessResults[0].Name = "mutated-after-write"
			Expect(persisted.Status.CurrentPeriod.Successful[0].Name).To(Equal("fresh-success"))
		})
	})

	Context("Retry-on-conflict when patching status", func() {
		It("retries once on a 409 and succeeds on the second Patch", func() {
			gvr := schema.GroupResource{Group: "kubecloudscaler.cloud", Resource: "k8s"}
			var calls int
			reconCtx.Client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(
						ctx context.Context,
						c client.Client,
						subResourceName string,
						obj client.Object,
						patch client.Patch,
						opts ...client.SubResourcePatchOption,
					) error {
						calls++
						if calls == 1 {
							return apierrors.NewConflict(gvr, obj.GetName(), fmt.Errorf("conflict"))
						}
						return c.Status().Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(calls).To(Equal(2))
		})
	})

	Context("When the scaler is deleted between Fetch and the finalizer-remove Patch", func() {
		It("completes without error (cleanup is idempotent)", func() {
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})
			// Fake client WITHOUT the scaler — any Get inside patchRemoveFinalizer returns NotFound.
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()
			reconCtx.ShouldFinalize = true

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(BeZero())
		})
	})

	Context("Retry-on-conflict when removing the finalizer", func() {
		It("retries once on a 409 and succeeds on the second Patch", func() {
			controllerutil.AddFinalizer(scaler, handlers.ScalerFinalizer)
			now := metav1.Now()
			scaler.SetDeletionTimestamp(&now)
			gvr := schema.GroupResource{Group: "kubecloudscaler.cloud", Resource: "k8s"}
			var patchCalls int
			reconCtx.Client = fake.NewClientBuilder().
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
			reconCtx.Scaler = scaler
			reconCtx.ShouldFinalize = true

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(patchCalls).To(Equal(2))
		})
	})
})
