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
	"errors"
	"fmt"
)

// CriticalError indicates an error that prevents further reconciliation and requires immediate stop.
//
// Usage:
//   - Authentication failures
//   - Invalid configuration
//   - Resource not found (when required)
//
// Behavior:
//   - Handler returns CriticalError and does not call next.Execute()
//   - Chain execution stops immediately
//   - Controller returns error to controller-runtime (no requeue)
type CriticalError struct {
	Err error
}

// Error returns the error message.
func (e *CriticalError) Error() string {
	return fmt.Sprintf("critical error: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *CriticalError) Unwrap() error {
	return e.Err
}

// NewCriticalError creates a new CriticalError wrapping the given error.
func NewCriticalError(err error) error {
	return &CriticalError{Err: err}
}

// RecoverableError indicates an error that may be resolved with a retry/requeue.
//
// Usage:
//   - Temporary rate limits
//   - Transient network issues
//   - API update conflicts
//
// Behavior:
//   - Handler returns RecoverableError and does not call next.Execute()
//   - Chain execution stops
//   - Controller returns ctrl.Result with requeue delay
type RecoverableError struct {
	Err error
}

// Error returns the error message.
func (e *RecoverableError) Error() string {
	return fmt.Sprintf("recoverable error: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *RecoverableError) Unwrap() error {
	return e.Err
}

// NewRecoverableError creates a new RecoverableError wrapping the given error.
func NewRecoverableError(err error) error {
	return &RecoverableError{Err: err}
}

// IsCriticalError checks if the given error is a CriticalError.
func IsCriticalError(err error) bool {
	var criticalErr *CriticalError
	return errors.As(err, &criticalErr)
}

// IsRecoverableError checks if the given error is a RecoverableError.
func IsRecoverableError(err error) bool {
	var recoverableErr *RecoverableError
	return errors.As(err, &recoverableErr)
}
