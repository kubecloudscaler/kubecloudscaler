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

// Package testutil provides test helpers for the K8s controller service layer.
package testutil

import (
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
)

// MockHandler is a mock implementation of the Handler interface for testing.
type MockHandler struct {
	ExecuteFunc func(ctx *service.ReconciliationContext) error
	next        service.Handler
}

// Execute delegates to ExecuteFunc if set, otherwise passes to next handler.
func (m *MockHandler) Execute(ctx *service.ReconciliationContext) error {
	if m.ExecuteFunc != nil {
		if err := m.ExecuteFunc(ctx); err != nil {
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

// SetNext sets the next handler in the chain.
func (m *MockHandler) SetNext(next service.Handler) {
	m.next = next
}
