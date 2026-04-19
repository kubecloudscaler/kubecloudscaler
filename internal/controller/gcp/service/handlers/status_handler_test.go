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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

var _ = Describe("StatusHandler", func() {
	var (
		logger        zerolog.Logger
		scheme        *runtime.Scheme
		statusHandler service.Handler
		reconCtx      *service.ReconciliationContext
		scaler        *kubecloudscalerv1alpha3.Gcp
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")

		statusHandler = handlers.NewStatusHandler()
	})

	Context("When updating status with successful results", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
				SuccessResults: []common.ScalerStatusSuccess{
					{Name: "instance-1", Kind: "instance"},
				},
				FailedResults: []common.ScalerStatusFailed{},
			}
		})

		It("should update status successfully", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})

	})

	Context("When updating status with failed results", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				SuccessResults: []common.ScalerStatusSuccess{},
				FailedResults: []common.ScalerStatusFailed{
					{Name: "instance-2", Kind: "instance", Reason: "API error"},
				},
			}
		})

		It("should update status with failures", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})
	})

	Context("When handling finalizer cleanup", func() {
		BeforeEach(func() {
			// Add finalizer to scaler
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				ShouldFinalize: true, // Deletion in progress
			}
		})

		It("should remove finalizer", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When client Patch fails during finalizer removal", func() {
		BeforeEach(func() {
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})

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
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				ShouldFinalize: true,
			}
		})

		It("should return a recoverable error and set RequeueAfter", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(5 * time.Second))
		})
	})

	Context("When client Status Update fails", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						ctx context.Context,
						c client.Client,
						subResourceName string,
						obj client.Object,
						opts ...client.SubResourceUpdateOption,
					) error {
						return fmt.Errorf("status update failure")
					},
				}).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				SuccessResults: []common.ScalerStatusSuccess{},
				FailedResults:  []common.ScalerStatusFailed{},
			}
		})

		It("should return a recoverable error and set RequeueAfter", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(5 * time.Second))
		})
	})

	Context("DeepCopy isolation between snapshot and Get-overwritten scaler", func() {
		It("writes the ctx-sourced Successful/Failed slices, not the server-side state", func() {
			// Pre-populate the server-side Status with different slice contents than what the
			// handler will write from ctx — a shallow *scaler.Status.CurrentPeriod copy would
			// alias the Successful/Failed slices and surface server state after the Get.
			serverScaler := scaler.DeepCopy()
			serverScaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{
				Name:       "prior-period",
				Successful: []common.ScalerStatusSuccess{{Name: "stale-from-server", Kind: "instance"}},
				Failed:     []common.ScalerStatusFailed{{Name: "stale-failed", Kind: "instance", Reason: "old"}},
			}
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(serverScaler).
				WithStatusSubresource(serverScaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
				SuccessResults: []common.ScalerStatusSuccess{
					{Name: "fresh-success", Kind: "instance"},
				},
				FailedResults: []common.ScalerStatusFailed{},
			}

			Expect(statusHandler.Execute(reconCtx)).To(Succeed())

			persisted := &kubecloudscalerv1alpha3.Gcp{}
			Expect(k8sClient.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(persisted.Status.CurrentPeriod).ToNot(BeNil())
			Expect(persisted.Status.CurrentPeriod.Successful).To(ConsistOf(
				common.ScalerStatusSuccess{Name: "fresh-success", Kind: "instance"},
			))
			Expect(persisted.Status.CurrentPeriod.Failed).To(BeEmpty())

			// Mutate the ctx source after the write — persisted data must not shift (no alias).
			reconCtx.SuccessResults[0].Name = "mutated-after-write"
			Expect(persisted.Status.CurrentPeriod.Successful[0].Name).To(Equal("fresh-success"))
		})
	})

	Context("Retry-on-conflict when updating status", func() {
		It("retries once on a 409 and succeeds on the second Update", func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func() func(context.Context, client.Client, string, client.Object, ...client.SubResourceUpdateOption) error {
						var calls int
						gvr := schema.GroupResource{Group: "kubecloudscaler.cloud", Resource: "gcps"}
						return func(ctx context.Context, c client.Client, _ string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
							calls++
							if calls == 1 {
								return apierrors.NewConflict(gvr, obj.GetName(), fmt.Errorf("conflict"))
							}
							return c.Status().Update(ctx, obj, opts...)
						}
					}(),
				}).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				SuccessResults: []common.ScalerStatusSuccess{{Name: "ok", Kind: "instance"}},
				FailedResults:  []common.ScalerStatusFailed{},
			}

			Expect(statusHandler.Execute(reconCtx)).To(Succeed())

			persisted := &kubecloudscalerv1alpha3.Gcp{}
			Expect(k8sClient.Get(reconCtx.Ctx, reconCtx.Request.NamespacedName, persisted)).To(Succeed())
			Expect(persisted.Status.CurrentPeriod.Successful).To(ConsistOf(
				common.ScalerStatusSuccess{Name: "ok", Kind: "instance"},
			))
		})
	})

	Context("When the scaler is deleted between Fetch and the finalizer-remove Patch", func() {
		It("completes without error (cleanup is idempotent)", func() {
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})
			// Fake client WITHOUT the scaler — any Get inside patchRemoveFinalizer returns NotFound.
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				ShouldFinalize: true,
			}

			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(BeZero())
		})
	})

	Context("Retry-on-conflict when removing the finalizer", func() {
		It("retries once on a 409 and succeeds on the second Patch", func() {
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})
			now := metav1.Now()
			scaler.SetDeletionTimestamp(&now)

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
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				ShouldFinalize: true,
			}

			Expect(statusHandler.Execute(reconCtx)).To(Succeed())
			Expect(patchCalls).To(Equal(2))
		})
	})

	Context("When both success and failed results exist", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
				SuccessResults: []common.ScalerStatusSuccess{
					{Name: "instance-1", Kind: "instance"},
					{Name: "instance-3", Kind: "instance"},
				},
				FailedResults: []common.ScalerStatusFailed{
					{Name: "instance-2", Kind: "instance", Reason: "timeout"},
				},
			}
		})

		It("should update status with both success and failures", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})
	})
})
