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

// Package clients provides interface definitions for Kubernetes client management.
package clients

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ConfigBuilder defines the interface for building Kubernetes configurations
type ConfigBuilder interface {
	BuildFromSecret(secret *corev1.Secret) (*rest.Config, error)
	BuildFromEnvironment() (*rest.Config, error)
	BuildFromKubeconfig(kubeconfigPath string) (*rest.Config, error)
}

// ClientFactory defines the interface for creating Kubernetes clients
type ClientFactory interface {
	CreateClient(config *rest.Config) (*kubernetes.Clientset, error)
}

// EnvironmentProvider defines the interface for environment operations
type EnvironmentProvider interface {
	GetEnv(key string) string
	FileExists(path string) bool
	ReadFile(path string) ([]byte, error)
}

// SecretValidator defines the interface for validating secrets
type SecretValidator interface {
	ValidateSecret(secret *corev1.Secret) error
}

// ClientManager defines the interface for managing Kubernetes clients
type ClientManager interface {
	GetClient(secret *corev1.Secret) (*kubernetes.Clientset, error)
}
