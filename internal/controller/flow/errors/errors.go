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

package errors

import (
	"fmt"
	"time"
)

// FlowError represents a flow processing error
type FlowError struct {
	Type    string
	Message string
	Err     error
}

func (e *FlowError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *FlowError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s (value: %s): %s", e.Field, e.Value, e.Message)
}

// SchedulingError represents a scheduling error with requeue information
type SchedulingError struct {
	Message      string
	RequeueAfter time.Duration
}

func (e *SchedulingError) Error() string {
	return fmt.Sprintf("scheduled for %s, requeue after %v", e.Message, e.RequeueAfter)
}

// ResourceError represents a resource-related error
type ResourceError struct {
	ResourceName string
	ResourceType string
	Operation    string
	Err          error
}

func (e *ResourceError) Error() string {
	return fmt.Sprintf("resource error for %s %s during %s: %v", e.ResourceType, e.ResourceName, e.Operation, e.Err)
}

func (e *ResourceError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewSchedulingError creates a new scheduling error
func NewSchedulingError(message string, requeueAfter time.Duration) *SchedulingError {
	return &SchedulingError{
		Message:      message,
		RequeueAfter: requeueAfter,
	}
}

// NewResourceError creates a new resource error
func NewResourceError(resourceName, resourceType, operation string, err error) *ResourceError {
	return &ResourceError{
		ResourceName: resourceName,
		ResourceType: resourceType,
		Operation:    operation,
		Err:          err,
	}
}

// NewFlowError creates a new flow error
func NewFlowError(errorType, message string, err error) *FlowError {
	return &FlowError{
		Type:    errorType,
		Message: message,
		Err:     err,
	}
}
