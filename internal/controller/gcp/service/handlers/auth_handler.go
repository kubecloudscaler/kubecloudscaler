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
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
)

// AuthHandler sets up GCP client with authentication.
type AuthHandler struct {
	next service.Handler
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler() service.Handler {
	return &AuthHandler{}
}

// Execute sets up GCP authentication and initializes the API client.
func (h *AuthHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := ctx.Scaler
	var secret *corev1.Secret

	if scaler.Spec.Config.AuthSecret != nil {
		secret = &corev1.Secret{}
		secretNamespace := os.Getenv("POD_NAMESPACE")
		if secretNamespace == "" {
			secretNamespace = "kubecloudscaler-system"
		}
		namespacedSecret := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      *scaler.Spec.Config.AuthSecret,
		}

		if err := ctx.Client.Get(ctx.Ctx, namespacedSecret, secret); err != nil {
			ctx.Logger.Error().Err(err).Str("secret", *scaler.Spec.Config.AuthSecret).Msg("fetch auth secret failed")
			return service.NewCriticalError(err)
		}
		ctx.Secret = secret
	}

	client, err := gcpClient.GetClient(secret, scaler.Spec.Config.ProjectID)
	if err != nil {
		ctx.Logger.Error().Err(err).Msg("create GCP client failed")
		return service.NewCriticalError(err)
	}

	ctx.GCPClient = client

	if h.next != nil {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *AuthHandler) SetNext(next service.Handler) {
	h.next = next
}
