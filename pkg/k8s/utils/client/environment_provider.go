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

// Package clients provides environment provider functionality for Kubernetes clients.
package clients

import (
	"os"
)

// environmentProvider implements EnvironmentProvider interface
type environmentProvider struct{}

// NewEnvironmentProvider creates a new environment provider
func NewEnvironmentProvider() EnvironmentProvider {
	return &environmentProvider{}
}

// GetEnv gets an environment variable
func (ep *environmentProvider) GetEnv(key string) string {
	return os.Getenv(key)
}

// FileExists checks if a file exists
func (ep *environmentProvider) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ReadFile reads a file
func (ep *environmentProvider) ReadFile(path string) ([]byte, error) {
	//nolint:gosec // G304: File path comes from trusted kubeconfig environment variable
	return os.ReadFile(path)
}
