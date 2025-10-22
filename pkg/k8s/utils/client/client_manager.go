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

// Package clients provides Kubernetes client management functionality.
package clients

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// clientManager implements ClientManager interface
type clientManager struct {
	configBuilder ConfigBuilder
	clientFactory ClientFactory
}

// NewClientManager creates a new client manager
func NewClientManager(configBuilder ConfigBuilder, clientFactory ClientFactory) ClientManager {
	return &clientManager{
		configBuilder: configBuilder,
		clientFactory: clientFactory,
	}
}

// GetClient returns a kubernetes clientset
func (cm *clientManager) GetClient(secret *corev1.Secret) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if secret != nil {
		config, err = cm.configBuilder.BuildFromSecret(secret)
	} else {
		config, err = cm.configBuilder.BuildFromEnvironment()
	}

	if err != nil {
		return nil, fmt.Errorf("error building config: %w", err)
	}

	clientset, err := cm.clientFactory.CreateClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	return clientset, nil
}
