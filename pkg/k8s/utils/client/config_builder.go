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

// Package clients provides Kubernetes configuration building functionality.
package clients

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// configBuilder implements ConfigBuilder interface
type configBuilder struct {
	envProvider EnvironmentProvider
}

// NewConfigBuilder creates a new config builder
func NewConfigBuilder(envProvider EnvironmentProvider) ConfigBuilder {
	return &configBuilder{
		envProvider: envProvider,
	}
}

// BuildFromSecret builds a Kubernetes config from a secret
func (cb *configBuilder) BuildFromSecret(secret *corev1.Secret) (*rest.Config, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret cannot be nil")
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("secret data cannot be nil")
	}

	// Validate required fields
	if err := cb.validateSecretData(secret); err != nil {
		return nil, fmt.Errorf("invalid secret data: %w", err)
	}

	// Parse insecure flag
	insecure, err := strconv.ParseBool(string(secret.Data["insecure"]))
	if err != nil {
		return nil, fmt.Errorf("error parsing insecure flag: %w", err)
	}

	config := &rest.Config{
		Host:        string(secret.Data["URL"]),
		BearerToken: string(secret.Data[corev1.ServiceAccountTokenKey]),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   secret.Data[corev1.ServiceAccountRootCAKey],
			Insecure: insecure,
		},
	}

	return config, nil
}

// BuildFromEnvironment builds a Kubernetes config from environment
func (cb *configBuilder) BuildFromEnvironment() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig if available
	kubeconfigPath := cb.envProvider.GetEnv("KUBECONFIG")
	if kubeconfigPath != "" {
		return cb.BuildFromKubeconfig(kubeconfigPath)
	}

	return nil, fmt.Errorf("error getting in-cluster config: %w", err)
}

// BuildFromKubeconfig builds a Kubernetes config from kubeconfig file
func (cb *configBuilder) BuildFromKubeconfig(kubeconfigPath string) (*rest.Config, error) {
	if !cb.envProvider.FileExists(kubeconfigPath) {
		return nil, fmt.Errorf("kubeconfig file does not exist: %s", kubeconfigPath)
	}

	// Use clientcmd to build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error building config from kubeconfig: %w", err)
	}

	return config, nil
}

// validateSecretData validates that the secret contains required data
func (cb *configBuilder) validateSecretData(secret *corev1.Secret) error {
	requiredFields := []string{
		"URL",
		corev1.ServiceAccountTokenKey,
		corev1.ServiceAccountRootCAKey,
		"insecure",
	}

	for _, field := range requiredFields {
		if _, exists := secret.Data[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}
