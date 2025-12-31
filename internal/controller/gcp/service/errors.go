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

import "fmt"

// ErrorCategory defines the severity of errors for appropriate handling in the chain.
//
// Error Categories:
//
//   - CriticalError: Errors that indicate reconciliation cannot proceed
//     Examples: authentication failures, invalid configuration, resource not found
//     Handling: Stop chain execution immediately
//
//   - RecoverableError: Errors that may be resolved with retry
//     Examples: temporary rate limits, transient network issues, temporary API unavailability
//     Handling: Allow chain continuation with requeue
//
// See data-model.md for full specification.
type ErrorCategory string

const (
	// CriticalError indicates an error that requires chain execution to stop.
	// Examples: authentication failures, invalid configuration, resource not found
	CriticalError ErrorCategory = "critical"

	// RecoverableError indicates an error that can be retried.
	// Examples: temporary rate limits, transient network issues, temporary API unavailability
	RecoverableError ErrorCategory = "recoverable"
)

// CriticalErr wraps an error as a critical error.
// Critical errors stop chain execution immediately.
type CriticalErr struct {
	Err error
}

// Error implements the error interface.
func (e *CriticalErr) Error() string {
	return fmt.Sprintf("critical error: %v", e.Err)
}

// Unwrap returns the wrapped error.
func (e *CriticalErr) Unwrap() error {
	return e.Err
}

// RecoverableErr wraps an error as a recoverable error.
// Recoverable errors allow chain continuation with requeue.
type RecoverableErr struct {
	Err error
}

// Error implements the error interface.
func (e *RecoverableErr) Error() string {
	return fmt.Sprintf("recoverable error: %v", e.Err)
}

// Unwrap returns the wrapped error.
func (e *RecoverableErr) Unwrap() error {
	return e.Err
}

// NewCriticalError creates a new critical error.
func NewCriticalError(err error) error {
	return &CriticalErr{Err: err}
}

// NewRecoverableError creates a new recoverable error.
func NewRecoverableError(err error) error {
	return &RecoverableErr{Err: err}
}

// IsCriticalError checks if an error is a critical error.
func IsCriticalError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*CriticalErr)
	return ok
}

// IsRecoverableError checks if an error is a recoverable error.
func IsRecoverableError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*RecoverableErr)
	return ok
}
