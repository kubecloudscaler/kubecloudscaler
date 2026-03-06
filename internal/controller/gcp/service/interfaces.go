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
// This follows the same pattern as the K8s controller:
//   - Processes its reconciliation step independently
//   - Modifies context as needed for subsequent handlers (RequeueAfter, SkipRemaining)
//   - Calls the next handler in the chain via Execute()
//   - Returns CriticalError to stop chain without requeue
//   - Returns RecoverableError to stop chain with requeue
//   - Returns nil to continue to the next handler
type Handler interface {
	// Execute processes a single reconciliation step.
	//
	// Returns:
	//   - error: nil to continue chain, CriticalError or RecoverableError to stop
	Execute(ctx *ReconciliationContext) error

	// SetNext sets the next handler in the chain.
	SetNext(next Handler)
}
