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
	"sync"
	"time"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespaceCacheTTL = 5 * time.Minute

// EnvProvider provides environment variable access (injectable for testability).
// clients.EnvironmentProvider from pkg/k8s/utils/client implements this interface.
type EnvProvider interface {
	GetEnv(key string) string
}

// defaultEnvProvider uses os.Getenv for production use.
type defaultEnvProvider struct{}

func (defaultEnvProvider) GetEnv(key string) string {
	return os.Getenv(key)
}

// namespaceManager implements NamespaceManager interface
type namespaceManager struct {
	client       KubernetesClient
	logger       zerolog.Logger
	envProvider  EnvProvider
	mu           sync.RWMutex
	cachedNsList []string
	cacheExpiry  time.Time
}

// NewNamespaceManager creates a new namespace manager.
// If envProvider is nil, a default provider using os.Getenv is used.
func NewNamespaceManager(client KubernetesClient, logger zerolog.Logger, envProvider EnvProvider) NamespaceManager {
	if envProvider == nil {
		envProvider = defaultEnvProvider{}
	}
	return &namespaceManager{
		client:      client,
		logger:      logger,
		envProvider: envProvider,
	}
}

// SetNamespaceList sets the namespace list based on configuration
func (nm *namespaceManager) SetNamespaceList(ctx context.Context, config *Config) ([]string, error) {
	var nsList []string

	if len(config.Namespaces) > 0 {
		nsList = config.Namespaces
	} else if cached := nm.readCachedNamespaces(config.ExcludeNamespaces); cached != nil {
		nsList = cached
	} else {
		var err error
		nsList, err = nm.refreshNamespaceCache(ctx, config.ExcludeNamespaces)
		if err != nil {
			return []string{}, err
		}
	}

	if config.ForceExcludeSystemNamespaces {
		nsList = nm.excludeSystemNamespaces(nsList)
	}

	nsList = nm.excludeOwnNamespace(nsList)

	return nsList, nil
}

func (nm *namespaceManager) readCachedNamespaces(excludeNamespaces []string) []string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if nm.cachedNsList == nil || !time.Now().Before(nm.cacheExpiry) {
		return nil
	}

	nsList := make([]string, 0, len(nm.cachedNsList))
	for _, ns := range nm.cachedNsList {
		if slices.Contains(excludeNamespaces, ns) {
			continue
		}
		nsList = append(nsList, ns)
	}
	return nsList
}

func (nm *namespaceManager) refreshNamespaceCache(ctx context.Context, excludeNamespaces []string) ([]string, error) {
	nsListItems, err := nm.client.CoreV1().Namespaces().List(ctx, metaV1.ListOptions{})
	if err != nil {
		nm.logger.Debug().Msg("error listing namespaces")
		return nil, fmt.Errorf("error listing namespaces: %w", err)
	}

	allNames := make([]string, 0, len(nsListItems.Items))

	for _, ns := range nsListItems.Items {
		allNames = append(allNames, ns.Name)
	}

	nm.mu.Lock()
	nm.cachedNsList = allNames
	nm.cacheExpiry = time.Now().Add(namespaceCacheTTL)
	nm.mu.Unlock()

	nsList := make([]string, 0, len(allNames))
	for _, ns := range allNames {
		if slices.Contains(excludeNamespaces, ns) {
			continue
		}
		nsList = append(nsList, ns)
	}
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
		Names:  config.Names,
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
	return slices.DeleteFunc(nsList, func(ns string) bool {
		return slices.Contains(DefaultExcludeNamespaces, ns)
	})
}

// excludeOwnNamespace removes the own namespace from the list
func (nm *namespaceManager) excludeOwnNamespace(nsList []string) []string {
	ownNamespace := nm.envProvider.GetEnv("POD_NAMESPACE")
	if ownNamespace == "" {
		return nsList
	}

	return slices.DeleteFunc(nsList, func(ns string) bool {
		return ns == ownNamespace
	})
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
