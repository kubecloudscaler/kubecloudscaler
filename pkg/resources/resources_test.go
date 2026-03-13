package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResource_UnknownResource(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	ctx := context.Background()

	tests := []struct {
		name         string
		resourceName string
	}{
		{name: "empty string", resourceName: ""},
		{name: "nonexistent resource", resourceName: "nonexistent"},
		{name: "typo in resource name", resourceName: "deployment"},
		{name: "uppercase variant", resourceName: "Deployments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource, err := NewResource(ctx, tt.resourceName, Config{}, &logger)
			require.Error(t, err)
			assert.Nil(t, resource)
			assert.True(t, errors.Is(err, ErrResourceNotFound), "expected ErrResourceNotFound, got: %v", err)
			assert.Contains(t, err.Error(), tt.resourceName)
		})
	}
}

func TestNewResource_K8sResourcesWithNilK8sConfig(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	ctx := context.Background()

	k8sResources := []string{
		"deployments",
		"statefulsets",
		"cronjobs",
		"github-ars",
	}

	for _, resourceName := range k8sResources {
		t.Run(resourceName, func(t *testing.T) {
			t.Parallel()

			config := Config{K8s: nil}
			resource, err := NewResource(ctx, resourceName, config, &logger)
			require.Error(t, err)
			assert.Nil(t, resource)
			assert.Contains(t, err.Error(), "K8s config is required")
		})
	}
}

func TestNewResource_GCPResourcesWithNilGCPConfig(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	ctx := context.Background()

	gcpResources := []string{
		"vm-instances",
	}

	for _, resourceName := range gcpResources {
		t.Run(resourceName, func(t *testing.T) {
			t.Parallel()

			config := Config{GCP: nil}
			resource, err := NewResource(ctx, resourceName, config, &logger)
			require.Error(t, err)
			assert.Nil(t, resource)
			assert.Contains(t, err.Error(), "GCP config is required")
		})
	}
}

func TestGetAvailableResources(t *testing.T) {
	t.Parallel()

	resources := GetAvailableResources()

	expected := []string{
		"deployments",
		"statefulsets",
		"cronjobs",
		"github-ars",
	}

	assert.Equal(t, expected, resources)
}

func TestGetAvailableResources_ReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	first := GetAvailableResources()
	// Mutate the returned slice
	first[0] = "mutated"

	second := GetAvailableResources()
	assert.Equal(t, "deployments", second[0], "mutation of returned slice should not affect the original")
}
