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

// Package service provides the service layer for K8s controller business logic.
// This package implements the Chain of Responsibility pattern using the classic
// refactoring.guru Go pattern where handlers have execute() and setNext() methods.
package service

// Handler defines the contract for all handlers in the Chain of Responsibility pattern.
// Each handler implements a single reconciliation step and maintains a reference to the next handler.
//
// This follows the classic Chain of Responsibility pattern from refactoring.guru:
// https://refactoring.guru/design-patterns/chain-of-responsibility/go/example
//
// Handlers must:
//   - Process their reconciliation step independently
//   - Modify context as needed for subsequent handlers
//   - Call next.execute() to pass control to the next handler (if not stopping)
//   - Return an error for critical failures that should stop the chain
//
// See contracts/handler-interface.md for full contract specification.
type Handler interface {
	// Execute processes a single reconciliation step.
	//
	// Parameters:
	//   - ctx: Reconciliation context containing shared state
	//
	// Returns:
	//   - error: Error encountered during execution (nil if successful)
	//
	// Behavior:
	//   - Handler processes its reconciliation step
	//   - If successful and should continue, calls next.Execute(ctx)
	//   - If error or should stop, returns without calling next
	//   - Handler should check if next is nil before calling next.Execute()
	//
	// Error Handling:
	//   - Critical errors: Return CriticalError, chain stops immediately
	//   - Recoverable errors: Return RecoverableError, allows requeue
	//   - Success: Return nil, continue with next handler
	Execute(ctx *ReconciliationContext) error

	// SetNext establishes the next handler in the chain.
	//
	// Parameters:
	//   - next: The next handler in the chain (can be nil to indicate end of chain)
	//
	// Behavior:
	//   - Sets the internal next handler reference
	//   - Can be called multiple times (last call wins)
	//   - Pass nil to indicate this is the last handler in the chain
	SetNext(next Handler)
}
