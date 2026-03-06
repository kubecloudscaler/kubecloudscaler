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

// CriticalErr wraps an error as a critical error.
// Critical errors stop chain execution immediately and should not be requeued.
type CriticalErr struct {
	Err error
}

func (e *CriticalErr) Error() string {
	return fmt.Sprintf("critical error: %v", e.Err)
}

func (e *CriticalErr) Unwrap() error {
	return e.Err
}

// RecoverableErr wraps an error as a recoverable error.
// Recoverable errors stop chain execution but are requeued.
type RecoverableErr struct {
	Err error
}

func (e *RecoverableErr) Error() string {
	return fmt.Sprintf("recoverable error: %v", e.Err)
}

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
