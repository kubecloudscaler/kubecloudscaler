package clients

import (
	// "k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClient returns a kubernetes clientset
// if secret is provided, it will use the secret to authenticate
// secret format:
// data:
//   "URL": "https://kubernetes.default.svc",
//   "token": "token",
//   "ca.crt": "ca.crt"

func GetClient(secret *corev1.Secret) (*kubernetes.Clientset, error) {
	var err error
	config := &rest.Config{}

	if secret != nil {
		insecure, err := strconv.ParseBool(string(secret.Data["insecure"]))
		if err != nil {
			return nil, fmt.Errorf("error parsing insecure: %w", err)
		}
		config = &rest.Config{
			Host:        string(secret.Data["URL"]),
			BearerToken: string(secret.Data[corev1.ServiceAccountTokenKey]),
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   secret.Data[corev1.ServiceAccountRootCAKey],
				Insecure: insecure,
			},
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			if os.Getenv("KUBECONFIG") != "" {
				config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
				if err != nil {
					return nil, fmt.Errorf("error building config from flags: %w", err)
				}
			} else {
				return nil, fmt.Errorf("error getting in-cluster config: %w", err)
			}
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %w", err)
	}

	return clientset, nil
}
