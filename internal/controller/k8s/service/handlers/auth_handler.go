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
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kubecloudscaler/kubecloudscaler/internal/config"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	k8sClient "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
)

// cachedClient holds a cached K8s client pair together with the ResourceVersion of the
// secret it was built from. When the secret is rotated (its ResourceVersion changes), the
// cached entry is treated as stale and the client is rebuilt — otherwise revoked
// credentials would remain usable until the controller restarts.
type cachedClient struct {
	resourceVersion string
	k8sClient       kubernetes.Interface
	dynamicClient   dynamic.Interface
}

// AuthHandler is a handler that sets up the K8s client with authentication.
type AuthHandler struct {
	next              service.Handler
	clientCache       sync.Map // map[string]*cachedClient (keyed by secret name, "" for default)
	namespaceResolver config.NamespaceResolver
}

// NewAuthHandler creates a new AuthHandler. If nsResolver is nil, uses config.DefaultNamespaceResolver().
func NewAuthHandler(nsResolver config.NamespaceResolver) service.Handler {
	if nsResolver == nil {
		nsResolver = config.DefaultNamespaceResolver()
	}
	return &AuthHandler{namespaceResolver: nsResolver}
}

// Execute sets up the K8s client with authentication and adds it to the reconciliation context.
//
// Behavior:
//   - If AuthSecret specified: Fetches secret, creates K8s client with secret
//   - If no AuthSecret: Creates K8s client with default credentials
//   - On success: Sets ctx.K8sClient, ctx.DynamicClient, ctx.Secret
func (h *AuthHandler) Execute(ctx *service.ReconciliationContext) error {
	// Handle authentication secret for K8s access
	var secret *corev1.Secret
	cacheKey := "" // default client
	if ctx.Scaler.Spec.Config.AuthSecret != nil {
		// Use operator namespace since K8s CRD is cluster-scoped (ctx.Request.Namespace is empty)
		secretNamespace := h.namespaceResolver.Resolve()
		namespacedSecret := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      *ctx.Scaler.Spec.Config.AuthSecret,
		}

		// Fetch the secret from the cluster
		secret = &corev1.Secret{}
		if err := ctx.Client.Get(ctx.Ctx, namespacedSecret, secret); err != nil {
			ctx.Logger.Error().Err(err).Msg("unable to fetch secret")
			return service.NewCriticalError(fmt.Errorf("unable to fetch auth secret: %w", err))
		}
		ctx.Secret = secret
		cacheKey = *ctx.Scaler.Spec.Config.AuthSecret
	} else {
		// No authentication secret specified, use default K8s access
		ctx.Secret = nil
		secret = nil
	}

	var secretRV string
	if secret != nil {
		secretRV = secret.ResourceVersion
	}

	// Cache lookup: hit only when the stored entry was built from the same ResourceVersion.
	// A mismatch means the secret was rotated since we cached the client, so we must rebuild.
	var kubeClient kubernetes.Interface
	var dynamicClient dynamic.Interface
	if cached, ok := h.clientCache.Load(cacheKey); ok {
		if cc, _ := cached.(*cachedClient); cc != nil && cc.resourceVersion == secretRV {
			kubeClient = cc.k8sClient
			dynamicClient = cc.dynamicClient
		}
	}

	if kubeClient == nil {
		var err error
		kubeClient, dynamicClient, err = k8sClient.GetClient(secret)
		if err != nil {
			ctx.Logger.Error().Err(err).Msg("unable to create K8s client")
			return service.NewCriticalError(fmt.Errorf("failed to create K8s client: %w", err))
		}
		h.clientCache.Store(cacheKey, &cachedClient{
			resourceVersion: secretRV,
			k8sClient:       kubeClient,
			dynamicClient:   dynamicClient,
		})
	}
	ctx.K8sClient = kubeClient
	ctx.DynamicClient = dynamicClient

	// Call next handler in chain
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *AuthHandler) SetNext(next service.Handler) {
	h.next = next
}
