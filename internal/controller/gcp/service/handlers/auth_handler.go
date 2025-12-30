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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
	ctrl "sigs.k8s.io/controller-runtime"
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
	next service.Handler
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler() service.Handler {
	return &AuthHandler{}
}

// Execute implements the Handler interface.
// It sets up GCP authentication and initializes the API client.
func (h *AuthHandler) Execute(req *service.ReconciliationContext) (ctrl.Result, error) {
	req.Logger.Debug().Msg("setting up GCP authentication")

	ctx := context.Background()
	scaler := req.Scaler
	var secret *corev1.Secret

	// Handle authentication secret if specified
	if scaler.Spec.Config.AuthSecret != nil {
		req.Logger.Info().Msg("fetching authentication secret")
		secret = &corev1.Secret{}
		namespacedSecret := types.NamespacedName{
			Namespace: scaler.Namespace,
			Name:      *scaler.Spec.Config.AuthSecret,
		}

		if err := req.Client.Get(ctx, namespacedSecret, secret); err != nil {
			req.Logger.Error().Err(err).Msg("unable to fetch authentication secret")
			return ctrl.Result{}, service.NewCriticalError(err)
		}
		req.Secret = secret
	}

	// Initialize GCP client
	client, err := gcpClient.GetClient(secret, scaler.Spec.Config.ProjectID)
	if err != nil {
		req.Logger.Error().Err(err).Msg("unable to create GCP client")
		return ctrl.Result{}, service.NewCriticalError(err)
	}

	req.GCPClient = client
	req.Logger.Info().Msg("GCP client initialized successfully")

	if h.next != nil {
		return h.next.Execute(req)
	}
	return ctrl.Result{}, nil
}

// SetNext sets the next handler in the chain.
func (h *AuthHandler) SetNext(next service.Handler) {
	h.next = next
}
