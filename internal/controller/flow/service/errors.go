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

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"
)

// Type aliases for backward compatibility — all error logic lives in shared package.
type CriticalError = shared.CriticalError
type RecoverableError = shared.RecoverableError

var (
	NewCriticalError    = shared.NewCriticalError
	NewRecoverableError = shared.NewRecoverableError
	IsCriticalError     = shared.IsCriticalError
	IsRecoverableError  = shared.IsRecoverableError
)

// ValidationReason is the closed set of reasons carried by a ValidationError. Values flow
// through to Flow.status as condition Reason strings, so they are a public contract and
// must remain stable — rename with care.
type ValidationReason string

const (
	ReasonUnknownPeriod             ValidationReason = "UnknownPeriod"
	ReasonInvalidPeriodDuration     ValidationReason = "InvalidPeriodDuration"
	ReasonZeroPeriodDuration        ValidationReason = "ZeroPeriodDuration"
	ReasonInvalidDelayFormat        ValidationReason = "InvalidDelayFormat"
	ReasonInvertedWindow            ValidationReason = "InvertedWindow"
	ReasonDuplicatePeriod           ValidationReason = "DuplicatePeriod"
	ReasonDuplicateResource         ValidationReason = "DuplicateResource"
	ReasonDuplicateResourceInPeriod ValidationReason = "DuplicateResourceInPeriod"
	ReasonAmbiguousResource         ValidationReason = "AmbiguousResource"
	ReasonUnknownResource           ValidationReason = "UnknownResource"
	ReasonUnknownResourceType       ValidationReason = "UnknownResourceType"
	ReasonMissingK8sResource        ValidationReason = "MissingK8sResource"
	ReasonMissingGcpResource        ValidationReason = "MissingGcpResource"
	// ReasonProcessingFailed is the fallback reason for non-validation errors that surface
	// on Flow.status. Kept here so the full set of condition reasons lives in one place.
	ReasonProcessingFailed ValidationReason = "ProcessingFailed"
)

// ValidationError marks a user-config error that will not be resolved by a retry. The Flow
// controller surfaces these as Kubernetes conditions with a specific Reason and classifies
// them as CriticalError so the reconcile does not hot-loop on a broken spec.
type ValidationError struct {
	Reason ValidationReason
	Err    error
}

// NewValidationError wraps err as a validation error with the given Reason. The Reason
// becomes the condition Reason on the Flow status.
func NewValidationError(reason ValidationReason, err error) error {
	return &ValidationError{Reason: reason, Err: err}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %v", e.Reason, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// IsValidationError reports whether err is (wraps) a *ValidationError.
func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// AsValidationError returns the first *ValidationError found while unwrapping err.
func AsValidationError(err error) (*ValidationError, bool) {
	var v *ValidationError
	if errors.As(err, &v) {
		return v, true
	}
	return nil, false
}
