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

	"github.com/kubecloudscaler/kubecloudscaler/internal/config"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
)

// AuthHandler sets up GCP client with authentication.
// This handler manages authentication secrets and initializes the GCP API client.
//
// Responsibilities:
//   - Fetch authentication secret if specified
//   - Initialize GCP client with credentials
//   - Populate GCPClient and Secret in context
//
// Error Handling:
//   - Secret not found: Critical error (stops chain)
//   - Client creation failure: Critical error (stops chain)
type AuthHandler struct {
	next              service.Handler
	namespaceResolver config.NamespaceResolver
}

// NewAuthHandler creates a new authentication handler. If nsResolver is nil, uses config.DefaultNamespaceResolver().
func NewAuthHandler(nsResolver config.NamespaceResolver) service.Handler {
	if nsResolver == nil {
		nsResolver = config.DefaultNamespaceResolver()
	}
	return &AuthHandler{namespaceResolver: nsResolver}
}

// Execute implements the Handler interface.
// It sets up GCP authentication and initializes the API client.
func (h *AuthHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := ctx.Scaler
	var secret *corev1.Secret

	if scaler.Spec.Config.AuthSecret != nil {
		secret = &corev1.Secret{}
		// Use operator namespace since GCP CRD is cluster-scoped (scaler.Namespace is empty)
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
	}

	// Initialize GCP client
	client, err := gcpClient.GetClient(ctx.Ctx, secret)
	if err != nil {
		ctx.Logger.Error().Err(err).Msg("unable to create GCP client")
		return service.NewCriticalError(fmt.Errorf("create GCP client: %w", err))
	}

	ctx.GCPClient = client
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *AuthHandler) SetNext(next service.Handler) {
	h.next = next
}
