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
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

const (
	testAuthSecretName = "gcp-secret"
	testGCPProjectID   = "test-project"
)

// stubNamespaceResolver returns a fixed namespace — keeps tests independent of POD_NAMESPACE.
type stubNamespaceResolver struct{ ns string }

func (s stubNamespaceResolver) Resolve() string { return s.ns }

// stubGCPFactory records invocations and returns a deterministic ClientSet (or error).
type stubGCPFactory struct {
	invocations []*corev1.Secret
	clientSet   *gcpUtils.ClientSet
	err         error
}

func (s *stubGCPFactory) build(_ context.Context, secret *corev1.Secret) (*gcpUtils.ClientSet, error) {
	s.invocations = append(s.invocations, secret)
	if s.err != nil {
		return nil, s.err
	}
	return s.clientSet, nil
}

func newStubGCPFactory() *stubGCPFactory {
	// Empty ClientSet is sufficient: the handler only passes it through to ctx.GCPClient.
	// No GCP API calls happen during AuthHandler.Execute.
	return &stubGCPFactory{clientSet: &gcpUtils.ClientSet{}}
}

var _ = Describe("AuthHandler", func() {
	var (
		logger      zerolog.Logger
		scheme      *runtime.Scheme
		authHandler service.Handler
		reconCtx    *service.ReconciliationContext
		scaler      *kubecloudscalerv1alpha3.Gcp
		factory     *stubGCPFactory
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")
		scaler.Spec.Config.ProjectID = testGCPProjectID

		factory = newStubGCPFactory()
		authHandler = handlers.NewAuthHandler(
			stubNamespaceResolver{ns: "default"},
			handlers.WithClientFactory(factory.build),
		)
	})

	Context("When auth secret is not specified", func() {
		BeforeEach(func() {
			scaler.Spec.Config.AuthSecret = nil
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should build a default-credentials client and expose it on the context", func() {
			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.Secret).To(BeNil())
			Expect(reconCtx.GCPClient).To(BeIdenticalTo(factory.clientSet))
			Expect(factory.invocations).To(HaveLen(1))
			Expect(factory.invocations[0]).To(BeNil())
		})

		It("should propagate factory errors as CriticalError", func() {
			factory.err = fmt.Errorf("ADC unreachable")

			err := authHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
			Expect(reconCtx.GCPClient).To(BeNil())
		})
	})

	Context("When auth secret is specified and exists", func() {
		var authSecret *corev1.Secret

		BeforeEach(func() {
			secretName := testAuthSecretName
			scaler.Spec.Config.AuthSecret = &secretName
			authSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretName,
					Namespace:       "default",
					ResourceVersion: "1",
				},
				Data: map[string][]byte{"service-account-key.json": []byte(`{"type":"service_account"}`)},
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler, authSecret).Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should fetch the secret and build the client", func() {
			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.Secret.Name).To(Equal(authSecret.Name))
			Expect(reconCtx.GCPClient).To(BeIdenticalTo(factory.clientSet))
			Expect(factory.invocations).To(HaveLen(1))
		})

		It("should reuse the cached client when the secret ResourceVersion is unchanged", func() {
			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(authHandler.Execute(reconCtx)).To(Succeed())

			// Second reconciliation must hit the cache, so factory is invoked only once.
			Expect(factory.invocations).To(HaveLen(1))
		})

		It("should rebuild the client when the secret is rotated (ResourceVersion changes)", func() {
			Expect(authHandler.Execute(reconCtx)).To(Succeed())

			// Simulate rotation: fetch latest, mutate, update — fake client bumps ResourceVersion.
			rotated := &corev1.Secret{}
			secretKey := types.NamespacedName{Namespace: authSecret.Namespace, Name: authSecret.Name}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, secretKey, rotated)).To(Succeed())
			rotated.Data["service-account-key.json"] = []byte(`{"type":"rotated"}`)
			Expect(reconCtx.Client.Update(reconCtx.Ctx, rotated)).To(Succeed())

			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(factory.invocations).To(HaveLen(2))
		})
	})

	Context("When the secret is rotated", func() {
		var (
			authSecret *corev1.Secret
			closes     atomic.Int32
		)

		BeforeEach(func() {
			secretName := testAuthSecretName
			scaler.Spec.Config.AuthSecret = &secretName
			authSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretName,
					Namespace:       "default",
					ResourceVersion: "1",
				},
				Data: map[string][]byte{"service-account-key.json": []byte(`{"type":"service_account"}`)},
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler, authSecret).Build()

			closes.Store(0)
			factory = newStubGCPFactory()
			authHandler = handlers.NewAuthHandler(
				stubNamespaceResolver{ns: "default"},
				handlers.WithClientFactory(factory.build),
				handlers.WithClientCloserForTest(func(_ *gcpUtils.ClientSet) error {
					closes.Add(1)
					return nil
				}),
			)

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("closes the stale client exactly once before storing the rebuilt client", func() {
			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(closes.Load()).To(BeZero(), "no close on initial build")

			rotated := &corev1.Secret{}
			secretKey := types.NamespacedName{Namespace: authSecret.Namespace, Name: authSecret.Name}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, secretKey, rotated)).To(Succeed())
			rotated.Data["service-account-key.json"] = []byte(`{"type":"rotated"}`)
			Expect(reconCtx.Client.Update(reconCtx.Ctx, rotated)).To(Succeed())

			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(factory.invocations).To(HaveLen(2))
			Expect(closes.Load()).To(Equal(int32(1)), "close called exactly once on rotation")

			// Third reconcile at the same RV must be a cache hit — no extra close, no extra build.
			Expect(authHandler.Execute(reconCtx)).To(Succeed())
			Expect(factory.invocations).To(HaveLen(2))
			Expect(closes.Load()).To(Equal(int32(1)))
		})
	})

	Context("Concurrent reconciles hitting the same cacheKey", func() {
		It("is goroutine-safe (run with -race); factory invocations are bounded", func() {
			secretName := testAuthSecretName
			scaler.Spec.Config.AuthSecret = &secretName
			authSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretName,
					Namespace:       "default",
					ResourceVersion: "1",
				},
				Data: map[string][]byte{"service-account-key.json": []byte(`{"type":"service_account"}`)},
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler, authSecret).Build()

			const workers = 20
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					local := &service.ReconciliationContext{
						Ctx:     context.Background(),
						Request: ctrl.Request{},
						Client:  k8sClient,
						Logger:  &logger,
						Scaler:  scaler,
					}
					Expect(authHandler.Execute(local)).To(Succeed())
				})
			}
			wg.Wait()

			// Under the current single-mutex cache, concurrent misses may serialise but none should
			// exceed the worker count, and at least one build must have happened.
			Expect(factory.invocations).ToNot(BeEmpty())
			Expect(len(factory.invocations)).To(BeNumerically("<=", workers))
		})
	})

	Context("WithClientFactory with a nil factory", func() {
		It("panics immediately — mis-wired tests must fail loudly, not fall through to ADC", func() {
			Expect(func() { handlers.WithClientFactory(nil) }).To(Panic())
		})
	})

	Context("When auth secret is specified but does not exist", func() {
		BeforeEach(func() {
			secretName := "nonexistent-secret"
			scaler.Spec.Config.AuthSecret = &secretName
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should return a critical error without invoking the factory", func() {
			err := authHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
			Expect(reconCtx.Secret).To(BeNil())
			Expect(factory.invocations).To(BeEmpty())
		})
	})
})
