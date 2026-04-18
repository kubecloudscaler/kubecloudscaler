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

package handlers

import (
	"sync"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

// cacheKey identifies a cached GCP ClientSet. The zero value {"", ""} denotes the
// Application-Default-Credentials path (no secret). Namespace is included so that two
// secrets with the same name in different namespaces never share a cached client — today
// the operator resolves a single namespace, but this defends against a future refactor that
// allows per-scaler secret namespaces.
type cacheKey struct {
	namespace string
	name      string
}

// clientCloser is the seam the cache uses to release a stale ClientSet on rotation. In
// production it delegates to *ClientSet.Close; tests inject a recording closer so that
// the close-before-store invariant is observable.
type clientCloser func(cs *gcpUtils.ClientSet) error

// cachedGCPClient pairs a ClientSet with the Secret.ResourceVersion it was built from.
// A rotation (RV change) invalidates the entry so revoked credentials cannot linger past
// the next reconcile.
type cachedGCPClient struct {
	resourceVersion string
	clientSet       *gcpUtils.ClientSet
}

// gcpClientCache is a goroutine-safe cache of GCP ClientSets keyed by (namespace, name) of
// the auth secret. All operations serialise through a single mutex; GCP auth setup is
// infrequent compared to the API calls downstream so per-key locking would be overkill.
//
// Concurrency caveat: Close is invoked synchronously when a stale entry is replaced. A
// long-running reconcile that captured a pointer via Get and is still using it when the
// rotation fires may observe a closed client. This is accepted: secret rotations are rare
// and gRPC surfaces the failure as a retryable error — preferable to leaking connections.
type gcpClientCache struct {
	mu      sync.Mutex
	entries map[cacheKey]*cachedGCPClient
	close   clientCloser
}

func newGCPClientCache() *gcpClientCache {
	return &gcpClientCache{
		entries: make(map[cacheKey]*cachedGCPClient),
		close:   func(cs *gcpUtils.ClientSet) error { return cs.Close() },
	}
}

// GetOrBuild returns a cached ClientSet whose stored ResourceVersion equals rv, or calls
// build to create one and stores it under key. On rotation (key present with a different
// rv) the previously cached ClientSet is closed before the new one is stored. The close
// error, if any, is returned alongside the new client so the caller can log it — the new
// client is always stored regardless. Concurrent calls are serialised.
func (c *gcpClientCache) GetOrBuild(
	key cacheKey,
	rv string,
	build func() (*gcpUtils.ClientSet, error),
) (*gcpUtils.ClientSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok && entry.resourceVersion == rv {
		return entry.clientSet, nil
	}

	cs, err := build()
	if err != nil {
		return nil, err
	}

	var closeErr error
	if prior, ok := c.entries[key]; ok {
		closeErr = c.close(prior.clientSet)
	}
	c.entries[key] = &cachedGCPClient{resourceVersion: rv, clientSet: cs}
	return cs, closeErr
}
