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

package clients

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// clientFactory implements ClientFactory interface
type clientFactory struct{}

// NewClientFactory creates a new client factory
func NewClientFactory() ClientFactory {
	return &clientFactory{}
}

// CreateClient creates a Kubernetes client from config
func (cf *clientFactory) CreateClient(config *rest.Config) (*kubernetes.Clientset, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %w", err)
	}

	return clientset, nil
}
