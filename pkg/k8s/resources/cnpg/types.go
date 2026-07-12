// Package cnpg provides type definitions for CloudNativePG Cluster resource management.
package cnpg

import (
	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// Cnpg represents a CloudNativePG Cluster resource manager.
type Cnpg struct {
	Resource          *utils.K8sResource
	Client            dynamic.NamespaceableResourceInterface
	Logger            *zerolog.Logger
	AnnotationManager utils.AnnotationManager
}
