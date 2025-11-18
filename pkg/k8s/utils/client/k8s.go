// Package clients provides Kubernetes client creation functionality.
package clients

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GetClient returns a kubernetes clientset and dynamic client using the new architecture
// This is a convenience function that maintains backward compatibility
func GetClient(secret *corev1.Secret) (*kubernetes.Clientset, dynamic.Interface, error) {
	// Create the new client manager with dependencies
	envProvider := NewEnvironmentProvider()
	configBuilder := NewConfigBuilder(envProvider)
	clientFactory := NewClientFactory()
	clientManager := NewClientManager(configBuilder, clientFactory)

	return clientManager.GetClient(secret)
}
