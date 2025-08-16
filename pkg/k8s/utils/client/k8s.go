package clients

import (
	// "k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
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

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %w", err)
	}

	return clientset, nil
}
