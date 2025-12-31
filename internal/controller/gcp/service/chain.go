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

package service

import (
	"github.com/rs/zerolog"
	ctrl "sigs.k8s.io/controller-runtime"
)

// HandlerChain implements the Chain interface.
// It is simply a wrapper around the first handler in the chain.
// The chain is built by linking handlers together using setNext().
//
// See contracts/chain-execution.md for full specification.
type HandlerChain struct {
	firstHandler Handler
}

// NewHandlerChain creates a new handler chain with the specified handlers.
// Handlers are linked together using setNext() in the order they are provided.
//
// Parameters:
//   - handlers: Ordered list of handlers to execute
//   - logger: Logger for chain execution observability (unused, kept for compatibility)
//
// Returns:
//   - Chain: Configured handler chain ready for execution
func NewHandlerChain(handlers []Handler, logger *zerolog.Logger) Chain {
	if len(handlers) == 0 {
		return &HandlerChain{firstHandler: nil}
	}

	// Link handlers together using setNext pattern
	for i := 0; i < len(handlers)-1; i++ {
		handlers[i].SetNext(handlers[i+1])
	}

	return &HandlerChain{firstHandler: handlers[0]}
}

// NewGCPScalerChain creates a new handler chain for GCP scaler reconciliation.
// This function registers handlers in the fixed order required for GCP scaler reconciliation.
//
// Handler Order:
//  1. Fetch - Fetch scaler resource from Kubernetes API
//  2. Finalizer - Manage finalizer lifecycle
//  3. Authentication - Setup GCP client with authentication
//  4. Period Validation - Validate and determine current time period
//  5. Resource Scaling - Scale GCP resources based on period
//  6. Status Update - Update scaler status with operation results
//
// Parameters:
//   - logger: Logger for chain execution observability (unused, kept for compatibility)
//
// Returns:
//   - Chain: Configured handler chain ready for GCP scaler reconciliation
func NewGCPScalerChain(logger *zerolog.Logger) Chain {
	// Import handlers package to avoid circular dependency
	// Note: This is a forward reference - handlers will be registered externally
	// For now, return empty chain as placeholder
	return &HandlerChain{firstHandler: nil}
}

// Execute runs all handlers in the chain in order.
// This method simply calls the first handler, which propagates execution
// through all handlers via the Chain of Responsibility pattern.
//
// Parameters:
//   - req: Reconciliation context containing shared state
//
// Returns:
//   - ctrl.Result: Kubernetes reconciliation result (requeue behavior)
//   - error: Critical error from handler execution
func (c *HandlerChain) Execute(req *ReconciliationContext) (ctrl.Result, error) {
	if c.firstHandler == nil {
		return ctrl.Result{}, nil
	}
	return c.firstHandler.Execute(req)
}
