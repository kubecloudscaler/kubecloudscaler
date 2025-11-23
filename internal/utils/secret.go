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

package utils

import (
	"context"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchSecret fetches a Kubernetes secret by name and namespace.
// If the secret name is nil or empty, returns nil without error.
//
// Parameters:
//   - ctx: The context for the operation
//   - k8sClient: The Kubernetes client
//   - secretName: Optional pointer to the secret name
//   - namespace: The namespace where the secret exists
//   - logger: Logger for logging operations
//
// Returns:
//   - *corev1.Secret: The fetched secret, or nil if no secret name was provided
//   - error: Any error that occurred during the fetch operation
func FetchSecret(
	ctx context.Context,
	k8sClient client.Client,
	secretName *string,
	namespace string,
	logger *zerolog.Logger,
) (*corev1.Secret, error) {
	if secretName == nil || *secretName == "" {
		return nil, nil
	}

	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      *secretName,
	}

	if err := k8sClient.Get(ctx, namespacedName, secret); err != nil {
		logger.Error().Err(err).Str("secret", *secretName).Msg("unable to fetch secret")
		return nil, err
	}

	return secret, nil
}
