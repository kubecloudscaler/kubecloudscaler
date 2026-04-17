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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	kfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/testutil"
)

// stubNamespaceResolver returns a fixed namespace — keeps tests independent of POD_NAMESPACE.
type stubNamespaceResolver struct{ ns string }

func (s stubNamespaceResolver) Resolve() string { return s.ns }

// stubClientFactory records invocations and returns a deterministic client triple.
type stubClientFactory struct {
	invocations []*corev1.Secret
	kubeClient  kubernetes.Interface
	dynClient   dynamic.Interface
	err         error
}

func (s *stubClientFactory) build(secret *corev1.Secret) (kubernetes.Interface, dynamic.Interface, error) {
	s.invocations = append(s.invocations, secret)
	if s.err != nil {
		return nil, nil, s.err
	}
	return s.kubeClient, s.dynClient, nil
}

func newStubFactory() *stubClientFactory {
	return &stubClientFactory{
		kubeClient: kfake.NewSimpleClientset(),
		dynClient:  dynfake.NewSimpleDynamicClient(runtime.NewScheme()),
	}
}

var _ = Describe("AuthHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		scaler   *kubecloudscalerv1alpha3.K8s
		factory  *stubClientFactory
	)

	BeforeEach(func() {
		factory = newStubFactory()
		handler = handlers.NewAuthHandler(stubNamespaceResolver{ns: "default"}, handlers.WithClientFactory(factory.build))
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
			Spec: kubecloudscalerv1alpha3.K8sSpec{},
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

	Context("When no AuthSecret is specified", func() {
		It("should build a default-credentials client and expose it on the context", func() {
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Secret).To(BeNil())
			Expect(reconCtx.K8sClient).To(BeIdenticalTo(factory.kubeClient))
			Expect(reconCtx.DynamicClient).To(BeIdenticalTo(factory.dynClient))
			Expect(factory.invocations).To(HaveLen(1))
			Expect(factory.invocations[0]).To(BeNil())
		})

		It("should propagate factory errors as CriticalError", func() {
			factory.err = fmt.Errorf("kubeconfig unreachable")
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
		})
	})

	Context("When an AuthSecret is specified and exists", func() {
		var authSecret *corev1.Secret

		BeforeEach(func() {
			secretName := "k8s-secret"
			scaler.Spec.Config.AuthSecret = ptr.To(secretName)
			authSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretName,
					Namespace:       "default",
					ResourceVersion: "1",
				},
				Data: map[string][]byte{"kubeconfig": []byte("fake")},
			}
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler, authSecret).Build()
		})

		It("should fetch the secret, build the client, and chain to next", func() {
			nextCalled := false
			handler.SetNext(&testutil.MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					nextCalled = true
					return nil
				},
			})

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(reconCtx.Secret.Name).To(Equal(authSecret.Name))
			Expect(nextCalled).To(BeTrue())
			Expect(factory.invocations).To(HaveLen(1))
		})

		It("should reuse the cached client when the secret ResourceVersion is unchanged", func() {
			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(handler.Execute(reconCtx)).To(Succeed())

			// Second reconciliation must hit the cache, so factory is invoked only once
			Expect(factory.invocations).To(HaveLen(1))
		})

		It("should rebuild the client when the secret is rotated (ResourceVersion changes)", func() {
			Expect(handler.Execute(reconCtx)).To(Succeed())

			// Simulate rotation: fetch latest, mutate, update — fake client bumps ResourceVersion.
			rotated := &corev1.Secret{}
			secretKey := types.NamespacedName{Namespace: authSecret.Namespace, Name: authSecret.Name}
			Expect(reconCtx.Client.Get(reconCtx.Ctx, secretKey, rotated)).To(Succeed())
			rotated.Data["kubeconfig"] = []byte("rotated")
			Expect(reconCtx.Client.Update(reconCtx.Ctx, rotated)).To(Succeed())

			Expect(handler.Execute(reconCtx)).To(Succeed())
			Expect(factory.invocations).To(HaveLen(2))
		})
	})

	Context("When an AuthSecret is specified but does not exist", func() {
		It("should return a critical error without invoking the factory", func() {
			scaler.Spec.Config.AuthSecret = ptr.To("non-existent-secret")
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

			err := handler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
			Expect(reconCtx.Secret).To(BeNil())
			Expect(factory.invocations).To(BeEmpty())
		})
	})
})
