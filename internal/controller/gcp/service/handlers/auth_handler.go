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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubecloudscaler/kubecloudscaler/internal/config"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
)

// ClientFactory builds a GCP ClientSet from an optional auth secret.
// Nil secret means use Application Default Credentials.
type ClientFactory func(ctx context.Context, secret *corev1.Secret) (*gcpUtils.ClientSet, error)

// defaultClientFactory delegates to the concrete gcpClient.GetClient factory.
func defaultClientFactory(ctx context.Context, secret *corev1.Secret) (*gcpUtils.ClientSet, error) {
	return gcpClient.GetClient(ctx, secret)
}

// AuthHandlerOption configures an AuthHandler at construction time.
type AuthHandlerOption func(*AuthHandler)

// WithClientFactory overrides the default client factory. Intended for tests. Passing a
// nil factory is a programmer error and panics immediately — silently keeping the default
// would make a mis-wired test fail mysteriously when it hits real ADC instead of the stub.
func WithClientFactory(factory ClientFactory) AuthHandlerOption {
	if factory == nil {
		panic("handlers.WithClientFactory: factory must not be nil")
	}
	return func(h *AuthHandler) {
		h.clientFactory = factory
	}
}

// withClientCloser overrides the cache's Close seam. Package-private — only used by tests
// in this package via a small white-box helper. Production must use the default.
func withClientCloser(fn clientCloser) AuthHandlerOption {
	if fn == nil {
		panic("handlers.withClientCloser: fn must not be nil")
	}
	return func(h *AuthHandler) {
		h.clientCache.close = fn
	}
}

// AuthHandler sets up the GCP client with authentication.
// This handler manages authentication secrets and initializes the GCP API client.
//
// Responsibilities:
//   - Fetch authentication secret if specified
//   - Cache the GCP ClientSet keyed by (secret namespace, secret name) + ResourceVersion so
//     a rotation invalidates stale credentials, and successive reconciliations reuse the
//     same client
//   - Populate GCPClient and Secret in context
//
// Error Handling:
//   - Secret not found: Critical error (stops chain)
//   - Client creation failure: Critical error (stops chain)
type AuthHandler struct {
	next              service.Handler
	clientCache       *gcpClientCache
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
		clientCache:       newGCPClientCache(),
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
	var key cacheKey // zero value = ADC path

	if scaler.Spec.Config.AuthSecret != nil {
		secret = &corev1.Secret{}
		// GCP CRD is cluster-scoped; the secret namespace comes from the operator resolver.
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
		key = cacheKey{namespace: secretNamespace, name: *scaler.Spec.Config.AuthSecret}
	}

	var secretRV string
	if secret != nil {
		secretRV = secret.ResourceVersion
	}

	clientSet, err := h.clientCache.GetOrBuild(key, secretRV, func() (*gcpUtils.ClientSet, error) {
		return h.clientFactory(ctx.Ctx, secret)
	})
	if err != nil {
		// Either the factory failed, or the factory succeeded but closing the prior stale
		// client returned an error. Distinguish via the returned clientSet: if it's nil, the
		// factory failed and we surface a critical error; otherwise we log the close error
		// and proceed with the fresh client.
		if clientSet == nil {
			ctx.Logger.Error().Err(err).Msg("unable to create GCP client")
			return service.NewCriticalError(fmt.Errorf("create GCP client: %w", err))
		}
		ctx.Logger.Warn().Err(err).Msg("failed to close stale GCP client on rotation")
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
