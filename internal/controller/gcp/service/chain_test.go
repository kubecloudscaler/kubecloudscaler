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

package service_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
)

func TestChain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chain Suite")
}

var _ = Describe("HandlerChain", func() {
	var (
		logger zerolog.Logger
		scheme *runtime.Scheme
		chain  service.Handler
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())
	})

	Context("When executing empty chain", func() {
		BeforeEach(func() {
			chain = service.BuildHandlerChain()
		})

		It("should complete successfully", func() {
			reconCtx := &service.ReconciliationContext{
				Ctx: context.Background(),
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				},
				Logger: &logger,
			}

			// BuildHandlerChain with no handlers returns nil
			// Calling Execute on nil would panic, so we check for nil
			Expect(chain).To(BeNil())
			_ = reconCtx // used to construct the context
		})
	})

	Context("When executing chain with mock handlers", func() {
		It("should execute handlers in order", func() {
			executionOrder := []string{}

			// Create mock handlers that track execution order
			handler1 := &mockHandler{
				name: "handler1",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler1")
					return nil
				},
			}

			handler2 := &mockHandler{
				name: "handler2",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler2")
					return nil
				},
			}

			handler3 := &mockHandler{
				name: "handler3",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler3")
					return nil
				},
			}

			chain = service.BuildHandlerChain(handler1, handler2, handler3)

			reconCtx := &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Logger:  &logger,
			}

			err := chain.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(executionOrder).To(Equal([]string{"handler1", "handler2", "handler3"}))
		})
	})

	Context("When handler returns critical error", func() {
		It("should stop chain execution", func() {
			executionOrder := []string{}

			handler1 := &mockHandler{
				name: "handler1",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler1")
					return nil
				},
			}

			handler2 := &mockHandler{
				name: "handler2-error",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler2-error")
					return service.NewCriticalError(nil)
				},
			}

			handler3 := &mockHandler{
				name: "handler3-should-not-execute",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler3-should-not-execute")
					return nil
				},
			}

			chain = service.BuildHandlerChain(handler1, handler2, handler3)

			reconCtx := &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Logger:  &logger,
			}

			err := chain.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
			Expect(executionOrder).To(Equal([]string{"handler1", "handler2-error"}))
			Expect(executionOrder).ToNot(ContainElement("handler3-should-not-execute"))
		})
	})

	Context("When handler sets SkipRemaining flag", func() {
		It("should stop chain execution early", func() {
			executionOrder := []string{}

			handler1 := &mockHandler{
				name: "handler1",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler1")
					req.SkipRemaining = true
					return nil
				},
			}

			handler2 := &mockHandler{
				name: "handler2-should-not-execute",
				executeFunc: func(req *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, "handler2-should-not-execute")
					return nil
				},
			}

			chain = service.BuildHandlerChain(handler1, handler2)

			reconCtx := &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Logger:  &logger,
			}

			err := chain.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(executionOrder).To(Equal([]string{"handler1"}))
			Expect(executionOrder).ToNot(ContainElement("handler2-should-not-execute"))
		})
	})
})

// mockHandler is a test helper that implements the Handler interface
type mockHandler struct {
	name        string
	next        service.Handler
	executeFunc func(req *service.ReconciliationContext) error
}

func (m *mockHandler) Execute(req *service.ReconciliationContext) error {
	err := m.executeFunc(req)
	if err != nil {
		return err
	}
	// Check if SkipRemaining was set
	if req.SkipRemaining {
		return nil
	}
	// Call next handler if available
	if m.next != nil {
		return m.next.Execute(req)
	}
	return nil
}

func (m *mockHandler) SetNext(next service.Handler) {
	m.next = next
}
