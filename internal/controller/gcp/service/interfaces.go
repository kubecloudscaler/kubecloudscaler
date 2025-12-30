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

// Package service provides the service layer for GCP controller business logic.
package service

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// Handler defines the contract for all handlers in the Chain of Responsibility pattern.
// Each handler implements a single reconciliation step and can modify the shared context.
//
// This follows the Chain of Responsibility pattern where each handler:
// - Processes its reconciliation step independently
// - Modifies context as needed for subsequent handlers
// - Calls the next handler in the chain via execute()
// - Sets the next handler via setNext()
//
// See contracts/handler-interface.md for full contract specification.
type Handler interface {
	// Execute processes a single reconciliation step and passes control to the next handler.
	//
	// Parameters:
	//   - req: Reconciliation context containing shared state
	//
	// Returns:
	//   - ctrl.Result: Kubernetes reconciliation result (requeue behavior)
	//   - error: Critical error that should stop chain execution (nil if successful)
	//
	// Error Handling:
	//   - Critical errors: Return error, chain stops immediately
	//   - Recoverable errors: Continue chain with requeue
	//   - Success: Call next handler if available
	Execute(req *ReconciliationContext) (ctrl.Result, error)

	// SetNext sets the next handler in the chain.
	//
	// Parameters:
	//   - next: The next handler to execute after this one
	SetNext(next Handler)
}

// Chain defines the contract for executing a sequence of handlers.
// The chain is simply the first handler in the chain, which will propagate
// execution through all handlers via the Chain of Responsibility pattern.
//
// See contracts/chain-execution.md for full contract specification.
type Chain interface {
	// Execute runs all handlers in the chain in order.
	//
	// Parameters:
	//   - req: Reconciliation context containing shared state
	//
	// Returns:
	//   - ctrl.Result: Kubernetes reconciliation result (requeue behavior)
	//   - error: Critical error from handler execution
	//
	// Behavior:
	//   - Executes handlers in fixed order (fetch → finalizer → auth → period → scaling → status)
	//   - Stops on critical errors
	//   - Continues on recoverable errors with requeue
	//   - Respects skip flag to stop early
	Execute(req *ReconciliationContext) (ctrl.Result, error)
}
