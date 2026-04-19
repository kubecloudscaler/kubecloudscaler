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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kubecloudscaler/kubecloudscaler/internal/config"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	k8sClient "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
)

// ClientFactory builds typed and dynamic K8s clients from an optional auth secret.
// Nil secret means use in-cluster / default credentials.
type ClientFactory func(secret *corev1.Secret) (kube kubernetes.Interface, dyn dynamic.Interface, err error)

// defaultClientFactory adapts the concrete k8sClient.GetClient to the ClientFactory signature.
func defaultClientFactory(secret *corev1.Secret) (kube kubernetes.Interface, dyn dynamic.Interface, err error) {
	return k8sClient.GetClient(secret)
}

// AuthHandlerOption configures an AuthHandler at construction time.
type AuthHandlerOption func(*AuthHandler)

// WithClientFactory overrides the default client factory. Intended for tests. Passing a
// nil factory is a programmer error and panics immediately — silently keeping the default
// would make a mis-wired test fail mysteriously when it hits real in-cluster config instead
// of the stub.
func WithClientFactory(factory ClientFactory) AuthHandlerOption {
	if factory == nil {
		panic("handlers.WithClientFactory: factory must not be nil")
	}
	return func(h *AuthHandler) {
		h.clientFactory = factory
	}
}

// AuthHandler sets up the K8s client with authentication.
// This handler manages authentication secrets and initializes the K8s API clients.
//
// Responsibilities:
//   - Fetch authentication secret if specified
//   - Cache typed + dynamic clients keyed by (secret namespace, secret name) +
//     ResourceVersion so a rotation invalidates stale credentials, and successive
//     reconciliations reuse the same clients
//   - Populate K8sClient, DynamicClient, and Secret in context
type AuthHandler struct {
	next              service.Handler
	clientCache       *k8sClientCache
	namespaceResolver config.NamespaceResolver
	clientFactory     ClientFactory
}

// NewAuthHandler creates a new AuthHandler. If nsResolver is nil, uses config.DefaultNamespaceResolver().
// Additional options (e.g., WithClientFactory) can be supplied for testing.
func NewAuthHandler(nsResolver config.NamespaceResolver, opts ...AuthHandlerOption) service.Handler {
	if nsResolver == nil {
		nsResolver = config.DefaultNamespaceResolver()
	}
	h := &AuthHandler{
		namespaceResolver: nsResolver,
		clientCache:       newK8sClientCache(),
		clientFactory:     defaultClientFactory,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Execute sets up the K8s client with authentication and adds it to the reconciliation context.
//
// Behavior:
//   - If AuthSecret specified: Fetches secret, creates K8s client with secret
//   - If no AuthSecret: Creates K8s client with default credentials
//   - On success: Sets ctx.K8sClient, ctx.DynamicClient, ctx.Secret
func (h *AuthHandler) Execute(ctx *service.ReconciliationContext) error {
	var secret *corev1.Secret
	var key cacheKey // zero value = default in-cluster credentials

	if ctx.Scaler.Spec.Config.AuthSecret != nil {
		// K8s CRD is cluster-scoped; the secret namespace comes from the operator resolver.
		secretNamespace := h.namespaceResolver.Resolve()
		namespacedSecret := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      *ctx.Scaler.Spec.Config.AuthSecret,
		}
		secret = &corev1.Secret{}
		if err := ctx.Client.Get(ctx.Ctx, namespacedSecret, secret); err != nil {
			ctx.Logger.Error().Err(err).Msg("unable to fetch secret")
			return service.NewCriticalError(fmt.Errorf("unable to fetch auth secret: %w", err))
		}
		ctx.Secret = secret
		key = cacheKey{namespace: secretNamespace, name: *ctx.Scaler.Spec.Config.AuthSecret}
	} else {
		ctx.Secret = nil
	}

	var secretRV string
	if secret != nil {
		secretRV = secret.ResourceVersion
	}

	kube, dyn, err := h.clientCache.GetOrBuild(key, secretRV, func() (kubernetes.Interface, dynamic.Interface, error) {
		return h.clientFactory(secret)
	})
	if err != nil {
		ctx.Logger.Error().Err(err).Msg("unable to create K8s client")
		return service.NewCriticalError(fmt.Errorf("failed to create K8s client: %w", err))
	}

	ctx.K8sClient = kube
	ctx.DynamicClient = dyn

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *AuthHandler) SetNext(next service.Handler) {
	h.next = next
}
