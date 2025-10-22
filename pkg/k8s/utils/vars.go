//nolint:nolintlint,revive // package name 'utils' is acceptable for K8s utility functions
package utils

var (
	// DefaultExcludeNamespaces contains the default namespaces to exclude from scaling.
	DefaultExcludeNamespaces = []string{"kube-system"}
)
