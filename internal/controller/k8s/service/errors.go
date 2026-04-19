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

import "github.com/kubecloudscaler/kubecloudscaler/internal/controller/shared"

// CriticalError is a non-retryable reconciliation failure (alias to shared).
type CriticalError = shared.CriticalError

// RecoverableError is a retryable reconciliation failure (alias to shared).
type RecoverableError = shared.RecoverableError

var (
	NewCriticalError    = shared.NewCriticalError
	NewRecoverableError = shared.NewRecoverableError
	IsCriticalError     = shared.IsCriticalError
	IsRecoverableError  = shared.IsRecoverableError
)
