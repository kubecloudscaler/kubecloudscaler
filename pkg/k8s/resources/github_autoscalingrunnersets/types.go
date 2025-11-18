// Package ars provides type definitions for Github Autoscaling Runnersets resource management.
package ars

import (
	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

// GithubAutoscalingRunnersets represents a Github Autoscaling Runnerset resource manager.
type GithubAutoscalingRunnersets struct {
	Resource *utils.K8sResource
	Client   dynamic.NamespaceableResourceInterface
	Logger   *zerolog.Logger
}
