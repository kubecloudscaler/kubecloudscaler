// Package scaledobjects provides type definitions for KEDA ScaledObject resource management.
package scaledobjects

import (
	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// ScaledObjects represents a KEDA ScaledObject resource manager.
type ScaledObjects struct {
	Resource          *utils.K8sResource
	Client            dynamic.NamespaceableResourceInterface
	Logger            *zerolog.Logger
	AnnotationManager utils.AnnotationManager
}
