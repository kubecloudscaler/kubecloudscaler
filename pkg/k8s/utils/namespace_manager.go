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

// Package utils provides namespace management functionality for Kubernetes resources.
package utils

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// namespaceManager implements NamespaceManager interface
type namespaceManager struct {
	client KubernetesClient
	logger zerolog.Logger
}

// NewNamespaceManager creates a new namespace manager
func NewNamespaceManager(client KubernetesClient, logger zerolog.Logger) NamespaceManager {
	return &namespaceManager{
		client: client,
		logger: logger,
	}
}

// SetNamespaceList sets the namespace list based on configuration
func (nm *namespaceManager) SetNamespaceList(ctx context.Context, config *Config) ([]string, error) {
	nsList := []string{}

	// get the list of namespaces
	if len(config.Namespaces) > 0 {
		nsList = config.Namespaces
	} else {
		// get all namespaces from the cluster
		nsListItems, err := nm.client.CoreV1().Namespaces().List(ctx, metaV1.ListOptions{})
		if err != nil {
			nm.logger.Debug().Msg("error listing namespaces")
			return []string{}, fmt.Errorf("error listing namespaces: %w", err)
		}

		for _, ns := range nsListItems.Items {
			if slices.Contains(config.ExcludeNamespaces, ns.Name) {
				continue
			}
			nsList = append(nsList, ns.Name)
		}
	}

	// force exclude system namespaces
	if config.ForceExcludeSystemNamespaces {
		nsList = nm.excludeSystemNamespaces(nsList)
	}

	// force always exclude my own namespace
	nsList = nm.excludeOwnNamespace(nsList)

	return nsList, nil
}

// PrepareSearch prepares search parameters for Kubernetes resources
func (nm *namespaceManager) PrepareSearch(ctx context.Context, config *Config) ([]string, metaV1.ListOptions, error) {
	nsList, err := nm.SetNamespaceList(ctx, config)
	if err != nil {
		nm.logger.Error().Err(err).Msg("unable to set namespace list")
		return []string{}, metaV1.ListOptions{}, err
	}

	// set a default label selector to ignore resources with the label "kubecloudscaler/ignore"
	labelSelectors := metaV1.LabelSelector{
		MatchLabels: make(map[string]string),
		MatchExpressions: []metaV1.LabelSelectorRequirement{
			{
				Key:      AnnotationsPrefix + "/ignore",
				Operator: metaV1.LabelSelectorOpDoesNotExist,
			},
		},
	}

	if config.LabelSelector != nil {
		nm.logger.Debug().Msgf("labelSelector: %+v", config.LabelSelector)
		labelSelectors = nm.mergeLabelSelectors(labelSelectors, *config.LabelSelector)
	}

	listOptions := metaV1.ListOptions{
		LabelSelector: metaV1.FormatLabelSelector(&labelSelectors),
	}

	return nsList, listOptions, nil
}

// InitConfig initializes a K8sResource with the given configuration
func (nm *namespaceManager) InitConfig(ctx context.Context, config *Config) (*K8sResource, error) {
	resource := &K8sResource{
		Period: config.Period,
	}

	nsList, listOptions, err := nm.PrepareSearch(ctx, config)
	if err != nil {
		return nil, err
	}

	resource.NsList = nsList
	resource.ListOptions = listOptions

	return resource, nil
}

// excludeSystemNamespaces removes system namespaces from the list
func (nm *namespaceManager) excludeSystemNamespaces(nsList []string) []string {
	for _, ns := range DefaultExcludeNamespaces {
		for i, n := range nsList {
			if n == ns {
				nsList = append(nsList[:i], nsList[i+1:]...)
			}
		}
	}
	return nsList
}

// excludeOwnNamespace removes the own namespace from the list
func (nm *namespaceManager) excludeOwnNamespace(nsList []string) []string {
	ownNamespace := os.Getenv("POD_NAMESPACE")
	if ownNamespace == "" {
		return nsList
	}

	for i, n := range nsList {
		if n == ownNamespace {
			nsList = append(nsList[:i], nsList[i+1:]...)
		}
	}
	return nsList
}

// mergeLabelSelectors merges two label selectors
func (nm *namespaceManager) mergeLabelSelectors(defaultSelector, customSelector metaV1.LabelSelector) metaV1.LabelSelector {
	// Copy match labels from custom selector
	if customSelector.MatchLabels != nil {
		for k, v := range customSelector.MatchLabels {
			defaultSelector.MatchLabels[k] = v
		}
	}

	// Add match expressions from custom selector, excluding ignore expressions
	if customSelector.MatchExpressions != nil {
		for _, v := range customSelector.MatchExpressions {
			if v.Key == AnnotationsPrefix+"/ignore" {
				continue
			}
			defaultSelector.MatchExpressions = append(defaultSelector.MatchExpressions, v)
		}
	}

	return defaultSelector
}
