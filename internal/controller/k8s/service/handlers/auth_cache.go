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

// Package handlers implements the Chain of Responsibility handlers for the K8s controller
// reconciliation loop. Each handler owns a single step (fetch, finalizer, auth, period,
// scaling, status) and is wired together via SetNext.
package handlers

import (
	"sync"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// cacheKey identifies a cached K8s client pair. The zero value {"", ""} denotes the
// in-cluster / default-credentials path (no secret). Namespace is included so that two
// secrets with the same name in different namespaces never share a cache entry — today
// the operator resolves a single namespace, but this defends against a future refactor
// that allows per-scaler secret namespaces.
type cacheKey struct {
	namespace string
	name      string
}

// cachedClient pairs a typed + dynamic client with the Secret.ResourceVersion they were
// built from. A rotation (RV change) invalidates the entry so revoked credentials cannot
// linger past the next reconcile.
type cachedClient struct {
	resourceVersion string
	k8sClient       kubernetes.Interface
	dynamicClient   dynamic.Interface
}

// k8sClientCache is a goroutine-safe cache of K8s client pairs keyed by (namespace, name).
// All operations serialise through a single mutex; auth setup is infrequent compared to
// the API calls downstream so per-key locking would be overkill. Unlike the GCP cache
// there is no Close seam: kubernetes.Interface / dynamic.Interface do not expose one and
// client-go's shared HTTP transport is torn down by GC + idle-conn timeouts.
type k8sClientCache struct {
	mu      sync.Mutex
	entries map[cacheKey]*cachedClient
}

func newK8sClientCache() *k8sClientCache {
	return &k8sClientCache{entries: make(map[cacheKey]*cachedClient)}
}

// GetOrBuild returns a cached client pair whose stored ResourceVersion equals rv, or calls
// build to create one and stores it under key. Concurrent calls are serialised.
func (c *k8sClientCache) GetOrBuild(
	key cacheKey,
	rv string,
	build func() (kube kubernetes.Interface, dyn dynamic.Interface, err error),
) (kube kubernetes.Interface, dyn dynamic.Interface, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok && entry.resourceVersion == rv {
		return entry.k8sClient, entry.dynamicClient, nil
	}

	kube, dyn, err = build()
	if err != nil {
		return nil, nil, err
	}

	c.entries[key] = &cachedClient{resourceVersion: rv, k8sClient: kube, dynamicClient: dyn}
	return kube, dyn, nil
}
