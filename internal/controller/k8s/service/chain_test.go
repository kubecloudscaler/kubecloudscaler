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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8s Service Suite")
}

// MockHandler is a mock implementation of the Handler interface for testing.
type MockHandler struct {
	ExecuteFunc func(ctx *service.ReconciliationContext) error
	next        service.Handler
	executed    bool
	order       int
}

func (m *MockHandler) Execute(ctx *service.ReconciliationContext) error {
	m.executed = true
	if m.ExecuteFunc != nil {
		err := m.ExecuteFunc(ctx)
		if err != nil {
			return err
		}
	}
	if ctx.SkipRemaining {
		return nil
	}
	if m.next != nil {
		return m.next.Execute(ctx)
	}
	return nil
}

func (m *MockHandler) SetNext(next service.Handler) {
	m.next = next
}

var _ = Describe("Handler Chain Integration", func() {
	var (
		logger zerolog.Logger
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
	})

	Context("When executing a chain of handlers", func() {
		It("should execute handlers in correct order", func() {
			executionOrder := []int{}

			handler1 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 1)
					return nil
				},
			}
			handler2 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 2)
					return nil
				},
			}
			handler3 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 3)
					return nil
				},
			}

			// Link handlers via setNext()
			handler1.SetNext(handler2)
			handler2.SetNext(handler3)

			reconCtx := &service.ReconciliationContext{
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				},
				Client: fake.NewClientBuilder().Build(),
				Logger: &logger,
			}

			// Start chain execution from first handler
			err := handler1.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(executionOrder).To(Equal([]int{1, 2, 3}))
		})

		It("should stop chain on critical error", func() {
			executionOrder := []int{}

			handler1 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 1)
					return nil
				},
			}
			handler2 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 2)
					return service.NewCriticalError(nil)
				},
			}
			handler3 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 3)
					return nil
				},
			}

			handler1.SetNext(handler2)
			handler2.SetNext(handler3)

			reconCtx := &service.ReconciliationContext{
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				},
				Client: fake.NewClientBuilder().Build(),
				Logger: &logger,
			}

			err := handler1.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
			Expect(executionOrder).To(Equal([]int{1, 2})) // Handler 3 should not be called
		})

		It("should stop chain when SkipRemaining is set", func() {
			executionOrder := []int{}

			handler1 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 1)
					return nil
				},
			}
			handler2 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 2)
					ctx.SkipRemaining = true
					return nil
				},
			}
			handler3 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					executionOrder = append(executionOrder, 3)
					return nil
				},
			}

			handler1.SetNext(handler2)
			handler2.SetNext(handler3)

			reconCtx := &service.ReconciliationContext{
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				},
				Client: fake.NewClientBuilder().Build(),
				Logger: &logger,
			}

			err := handler1.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(executionOrder).To(Equal([]int{1, 2})) // Handler 3 should not be called
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})

		It("should allow context modification through chain", func() {
			handler1 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					ctx.ShouldFinalize = true
					return nil
				},
			}
			handler2 := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					// Verify handler1 modified context
					Expect(ctx.ShouldFinalize).To(BeTrue())
					return nil
				},
			}

			handler1.SetNext(handler2)

			reconCtx := &service.ReconciliationContext{
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				},
				Client: fake.NewClientBuilder().Build(),
				Logger: &logger,
			}

			err := handler1.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle empty chain (nil next)", func() {
			handler := &MockHandler{
				ExecuteFunc: func(ctx *service.ReconciliationContext) error {
					return nil
				},
			}
			// No next handler set

			reconCtx := &service.ReconciliationContext{
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				},
				Client: fake.NewClientBuilder().Build(),
				Logger: &logger,
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
		})
	})
})
