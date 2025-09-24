package client

import (
	"context"
	"fmt"
	"os"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"
)

// GetClient creates a GCP Compute Engine client
// It supports authentication via service account key from Kubernetes secret or default credentials
func GetClient(secret *corev1.Secret, projectId string) (*compute.Service, error) {
	var client *compute.Service
	var err error

	if secret != nil {
		// Use service account key from Kubernetes secret
		serviceAccountKey, exists := secret.Data["service-account-key.json"]
		if !exists {
			return nil, fmt.Errorf("service-account-key.json not found in secret")
		}

		// Create client with service account key
		client, err = compute.NewService(context.Background(), option.WithCredentialsJSON(serviceAccountKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP client with service account key: %w", err)
		}
	} else {
		// Use default credentials (Application Default Credentials)
		// This works when running in GKE or with gcloud auth application-default login
		client, err = compute.NewService(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP client with default credentials: %w", err)
		}
	}

	// Verify the client can access the project
	if projectId != "" {
		if err := verifyProjectAccess(client, projectId); err != nil {
			return nil, fmt.Errorf("failed to verify project access: %w", err)
		}
	}

	return client, nil
}

// verifyProjectAccess verifies that the client can access the specified project
func verifyProjectAccess(client *compute.Service, projectId string) error {
	// Try to get project information to verify access
	_, err := client.Projects.Get(projectId).Do()
	if err != nil {
		return fmt.Errorf("cannot access project %s: %w", projectId, err)
	}
	return nil
}

// GetProjectFromEnv returns the project ID from environment variable
func GetProjectFromEnv() string {
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}

// GetRegionFromEnv returns the region from environment variable
func GetRegionFromEnv() string {
	return os.Getenv("GOOGLE_CLOUD_REGION")
}
