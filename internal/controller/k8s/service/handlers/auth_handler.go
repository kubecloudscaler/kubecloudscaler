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
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	k8sClient "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
)

// AuthHandler is a handler that sets up the K8s client with authentication.
type AuthHandler struct {
	next service.Handler
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler() service.Handler {
	return &AuthHandler{}
}

// Execute sets up the K8s client with authentication and adds it to the reconciliation context.
//
// Behavior:
//   - If AuthSecret specified: Fetches secret, creates K8s client with secret
//   - If no AuthSecret: Creates K8s client with default credentials
//   - On success: Sets ctx.K8sClient, ctx.DynamicClient, ctx.Secret
func (h *AuthHandler) Execute(ctx *service.ReconciliationContext) error {
	ctx.Logger.Debug().Msg("setting up K8s authentication")

	// Handle authentication secret for K8s access
	var secret *corev1.Secret
	if ctx.Scaler.Spec.Config.AuthSecret != nil {
		ctx.Logger.Info().Msg("auth secret found for K8s authentication")
		// Construct the namespaced name for the secret
		namespacedSecret := types.NamespacedName{
			Namespace: ctx.Request.Namespace,
			Name:      *ctx.Scaler.Spec.Config.AuthSecret,
		}

		// Fetch the secret from the cluster
		secret = &corev1.Secret{}
		if err := ctx.Client.Get(context.Background(), namespacedSecret, secret); err != nil {
			ctx.Logger.Error().Err(err).Msg("unable to fetch secret")
			return service.NewCriticalError(errors.New("unable to fetch auth secret"))
		}
		ctx.Secret = secret
	} else {
		// No authentication secret specified, use default K8s access
		ctx.Secret = nil
		secret = nil
	}

	// Initialize K8s client for resource operations
	kubeClient, dynamicClient, err := k8sClient.GetClient(secret)
	if err != nil {
		ctx.Logger.Error().Err(err).Msg("unable to create K8s client")
		return service.NewCriticalError(errors.New("failed to create K8s client"))
	}
	ctx.K8sClient = kubeClient
	ctx.DynamicClient = dynamicClient

	ctx.Logger.Debug().Msg("K8s client initialized successfully")

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
