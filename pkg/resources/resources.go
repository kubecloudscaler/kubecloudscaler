// Package resources provides resource management functionality for the kubecloudscaler project.
package resources

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	vminstances "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/resources/vm-instances"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/cronjobs"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/deployments"
	ars "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/github_autoscalingrunnersets"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/statefulsets"
	// "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/horizontalpodautoscalers"
)

// NewResource creates a new resource instance based on the resource type name.
// It returns the created resource or an error if the resource type is invalid.
func NewResource(resourceName string, config Config, logger *zerolog.Logger) (IResource, error) {
	ctx := context.Background()
	ctx = logger.With().Str("resource-type", resourceName).Logger().WithContext(ctx)

	switch resourceName {
	case "deployments":
		return newDeploymentsResource(ctx, config)
	case "statefulsets":
		return newStatefulSetsResource(ctx, config)
	case "cronjobs":
		return newCronJobsResource(ctx, config)
	case "vm-instances":
		return newVMInstancesResource(ctx, config)
	case "github-ars":
		return newGitHubARSResource(ctx, config)
	default:
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, resourceName)
	}
}

// newDeploymentsResource creates a new deployments resource
func newDeploymentsResource(ctx context.Context, config Config) (IResource, error) {
	if config.K8s == nil {
		return nil, fmt.Errorf("K8s config is required for deployments resource")
	}
	resource, err := deployments.New(ctx, config.K8s)
	if err != nil {
		return nil, fmt.Errorf("error creating deployments resource: %w", err)
	}
	return resource, nil
}

// newStatefulSetsResource creates a new statefulsets resource
func newStatefulSetsResource(ctx context.Context, config Config) (IResource, error) {
	if config.K8s == nil {
		return nil, fmt.Errorf("K8s config is required for statefulsets resource")
	}
	resource, err := statefulsets.New(ctx, config.K8s)
	if err != nil {
		return nil, fmt.Errorf("error creating statefulsets resource: %w", err)
	}
	return resource, nil
}

// newCronJobsResource creates a new cronjobs resource
func newCronJobsResource(ctx context.Context, config Config) (IResource, error) {
	if config.K8s == nil {
		return nil, fmt.Errorf("K8s config is required for cronjobs resource")
	}
	resource, err := cronjobs.New(ctx, config.K8s)
	if err != nil {
		return nil, fmt.Errorf("error creating cronjobs resource: %w", err)
	}
	return resource, nil
}

// newVMInstancesResource creates a new vm-instances resource
func newVMInstancesResource(ctx context.Context, config Config) (IResource, error) {
	if config.GCP == nil {
		return nil, fmt.Errorf("GCP config is required for vm-instances resource")
	}
	resource, err := vminstances.New(ctx, config.GCP)
	if err != nil {
		return nil, fmt.Errorf("error creating vm-instances resource: %w", err)
	}
	return resource, nil
}

// newGitHubARSResource creates a new github-ars resource
func newGitHubARSResource(ctx context.Context, config Config) (IResource, error) {
	if config.K8s == nil {
		return nil, fmt.Errorf("K8s config is required for github-ars resource")
	}
	resource, err := ars.New(ctx, config.K8s)
	if err != nil {
		return nil, fmt.Errorf("error creating github-ars resource: %w", err)
	}
	return resource, nil
}
