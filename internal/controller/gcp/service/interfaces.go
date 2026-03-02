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

// Handler defines the contract for all handlers in the Chain of Responsibility pattern.
// Each handler implements a single reconciliation step and can modify the shared context.
//
// This follows the Chain of Responsibility pattern where each handler:
// - Processes its reconciliation step independently
// - Modifies context as needed for subsequent handlers
// - Calls the next handler in the chain via Execute()
// - Sets the next handler via SetNext()
//
// Error Handling:
//   - Critical errors: Return CriticalError, chain stops immediately, no requeue
//   - Recoverable errors: Return RecoverableError, chain stops, requeue with backoff
//   - Success: Return nil, continue with next handler
//
// Handlers communicate requeue timing via ReconciliationContext.RequeueAfter,
// not via ctrl.Result. The controller translates context state to ctrl.Result.
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
	Execute(ctx *ReconciliationContext) error

	// SetNext establishes the next handler in the chain.
	//
	// Parameters:
	//   - next: The next handler in the chain (can be nil to indicate end of chain)
	SetNext(next Handler)
}
