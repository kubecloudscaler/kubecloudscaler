package client

import (
	"context"
	"fmt"
	"os"

	compute "cloud.google.com/go/compute/apiv1"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

// GetClient creates a GCP Compute Engine client
// It supports authentication via service account key from Kubernetes secret or default credentials
func GetClient(secret *corev1.Secret, projectId string) (*gcpUtils.ClientSet, error) {
	var (
		instancesClient      *compute.InstancesClient
		regionsClient        *compute.RegionsClient
		zoneOperationsClient *compute.ZoneOperationsClient
		err                  error
	)

	ctx := context.Background()

	if secret != nil {
		// Use service account key from Kubernetes secret
		serviceAccountKey, exists := secret.Data["service-account-key.json"]
		if !exists {
			return nil, fmt.Errorf("service-account-key.json not found in secret")
		}

		instancesClient, err = compute.NewInstancesRESTClient(ctx, option.WithCredentialsJSON(serviceAccountKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create Instances client: %w", err)
		}

		regionsClient, err = compute.NewRegionsRESTClient(ctx, option.WithCredentialsJSON(serviceAccountKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create Regions client: %w", err)
		}

		zoneOperationsClient, err = compute.NewZoneOperationsRESTClient(ctx, option.WithCredentialsJSON(serviceAccountKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create ZoneOperations client: %w", err)
		}
	} else {
		// Use default credentials (Application Default Credentials)
		instancesClient, err = compute.NewInstancesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create Instances client with default credentials: %w", err)
		}

		regionsClient, err = compute.NewRegionsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create Regions client with default credentials: %w", err)
		}

		zoneOperationsClient, err = compute.NewZoneOperationsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ZoneOperations client with default credentials: %w", err)
		}
	}

	return &gcpUtils.ClientSet{
		Instances:      instancesClient,
		Regions:        regionsClient,
		ZoneOperations: zoneOperationsClient,
	}, nil
}

// GetProjectFromEnv returns the project ID from environment variable
func GetProjectFromEnv() string {
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}

// GetRegionFromEnv returns the region from environment variable
func GetRegionFromEnv() string {
	return os.Getenv("GOOGLE_CLOUD_REGION")
}
