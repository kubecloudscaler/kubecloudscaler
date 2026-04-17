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
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubecloudscaler/kubecloudscaler/internal/config"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
)

// cachedGCPClient holds a cached GCP client set together with the ResourceVersion of the
// secret it was built from. When the secret is rotated (its ResourceVersion changes), the
// cached entry is treated as stale and the client is rebuilt — otherwise revoked
// credentials would remain usable until the controller restarts.
type cachedGCPClient struct {
	resourceVersion string
	clientSet       *gcpUtils.ClientSet
}

// ClientFactory builds a GCP ClientSet from an optional auth secret.
// Nil secret means use Application Default Credentials.
type ClientFactory func(ctx context.Context, secret *corev1.Secret) (*gcpUtils.ClientSet, error)

// defaultClientFactory delegates to the concrete gcpClient.GetClient factory.
func defaultClientFactory(ctx context.Context, secret *corev1.Secret) (*gcpUtils.ClientSet, error) {
	return gcpClient.GetClient(ctx, secret)
}

// AuthHandlerOption configures an AuthHandler at construction time.
type AuthHandlerOption func(*AuthHandler)

// WithClientFactory overrides the default client factory. Intended for tests.
func WithClientFactory(factory ClientFactory) AuthHandlerOption {
	return func(h *AuthHandler) {
		if factory != nil {
			h.clientFactory = factory
		}
	}
}

// AuthHandler sets up the GCP client with authentication.
// This handler manages authentication secrets and initializes the GCP API client.
//
// Responsibilities:
//   - Fetch authentication secret if specified
//   - Cache the GCP ClientSet keyed by secret name + ResourceVersion so a rotation invalidates
//     stale credentials, and successive reconciliations reuse the same client
//   - Populate GCPClient and Secret in context
//
// Error Handling:
//   - Secret not found: Critical error (stops chain)
//   - Client creation failure: Critical error (stops chain)
type AuthHandler struct {
	next              service.Handler
	clientCache       sync.Map // map[string]*cachedGCPClient (key: secret name, "" for default)
	namespaceResolver config.NamespaceResolver
	clientFactory     ClientFactory
}

// NewAuthHandler creates a new authentication handler. If nsResolver is nil, uses
// config.DefaultNamespaceResolver(). Additional options (e.g., WithClientFactory) can be
// supplied for testing.
func NewAuthHandler(nsResolver config.NamespaceResolver, opts ...AuthHandlerOption) service.Handler {
	if nsResolver == nil {
		nsResolver = config.DefaultNamespaceResolver()
	}
	h := &AuthHandler{
		namespaceResolver: nsResolver,
		clientFactory:     defaultClientFactory,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Execute sets up GCP authentication and initializes the API client.
func (h *AuthHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := ctx.Scaler
	var secret *corev1.Secret
	cacheKey := "" // default credentials
	if scaler.Spec.Config.AuthSecret != nil {
		secret = &corev1.Secret{}
		// Use operator namespace since GCP CRD is cluster-scoped (scaler.Namespace is empty).
		secretNamespace := h.namespaceResolver.Resolve()
		namespacedSecret := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      *scaler.Spec.Config.AuthSecret,
		}

		if err := ctx.Client.Get(ctx.Ctx, namespacedSecret, secret); err != nil {
			ctx.Logger.Error().Err(err).Msg("unable to fetch authentication secret")
			return service.NewCriticalError(fmt.Errorf("fetch authentication secret: %w", err))
		}
		ctx.Secret = secret
		cacheKey = *scaler.Spec.Config.AuthSecret
	}

	var secretRV string
	if secret != nil {
		secretRV = secret.ResourceVersion
	}

	// Cache lookup: hit only when the stored entry was built from the same ResourceVersion.
	// On mismatch we close the stale client before rebuilding so we don't leak connections.
	var clientSet *gcpUtils.ClientSet
	if cached, ok := h.clientCache.Load(cacheKey); ok {
		if cc, _ := cached.(*cachedGCPClient); cc != nil {
			if cc.resourceVersion == secretRV {
				clientSet = cc.clientSet
			} else {
				if closeErr := cc.clientSet.Close(); closeErr != nil {
					ctx.Logger.Warn().Err(closeErr).Msg("failed to close stale GCP client on rotation")
				}
			}
		}
	}

	if clientSet == nil {
		var err error
		clientSet, err = h.clientFactory(ctx.Ctx, secret)
		if err != nil {
			ctx.Logger.Error().Err(err).Msg("unable to create GCP client")
			return service.NewCriticalError(fmt.Errorf("create GCP client: %w", err))
		}
		h.clientCache.Store(cacheKey, &cachedGCPClient{
			resourceVersion: secretRV,
			clientSet:       clientSet,
		})
	}

	ctx.GCPClient = clientSet
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *AuthHandler) SetNext(next service.Handler) {
	h.next = next
}
