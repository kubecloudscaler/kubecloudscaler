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

// Package config provides configuration management for the kubecloudscaler project.
package config

import "os"

// DefaultOperatorNamespace is the fallback namespace when POD_NAMESPACE is not set
// (e.g. when running locally outside a cluster).
const DefaultOperatorNamespace = "kubecloudscaler-system"

// PodNamespaceEnv is the environment variable containing the operator's pod namespace.
const PodNamespaceEnv = "POD_NAMESPACE"

// NamespaceResolver resolves the operator namespace for cluster-scoped CRDs.
// Defined at consumer site for testability (inject mock in tests).
type NamespaceResolver interface {
	Resolve() string
}

// EnvNamespaceResolver resolves namespace from environment variable with fallback.
type EnvNamespaceResolver struct {
	EnvKey    string
	DefaultNs string
	envLookup func(string) string
}

// NewEnvNamespaceResolver creates a resolver that reads from env with fallback.
func NewEnvNamespaceResolver(envKey, defaultNs string) *EnvNamespaceResolver {
	return &EnvNamespaceResolver{
		EnvKey:    envKey,
		DefaultNs: defaultNs,
		envLookup: os.Getenv,
	}
}

// Resolve returns the operator namespace.
func (r *EnvNamespaceResolver) Resolve() string {
	if ns := r.envLookup(r.EnvKey); ns != "" {
		return ns
	}
	return r.DefaultNs
}

// DefaultNamespaceResolver returns a resolver using POD_NAMESPACE env with DefaultOperatorNamespace fallback.
func DefaultNamespaceResolver() NamespaceResolver {
	return NewEnvNamespaceResolver(PodNamespaceEnv, DefaultOperatorNamespace)
}
