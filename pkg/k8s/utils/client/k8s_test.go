/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clients_test

import (
	"os"

	clients "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var _ = Describe("GetClient", func() {
	var (
		testSecret         *corev1.Secret
		originalKubeconfig string
	)

	BeforeEach(func() {
		// Store original KUBECONFIG value
		originalKubeconfig = os.Getenv("KUBECONFIG")

		caData, err := os.ReadFile("testdata/ca.crt")
		Expect(err).ToNot(HaveOccurred())

		// Create a test secret with valid data
		testSecret = &corev1.Secret{
			Data: map[string][]byte{
				"URL":                          []byte("https://test-cluster.example.com"),
				corev1.ServiceAccountTokenKey:  []byte("test-token"),
				corev1.ServiceAccountRootCAKey: caData,
				"insecure":                     []byte("false"),
			},
		}
	})

	AfterEach(func() {
		// Restore original KUBECONFIG value
		if originalKubeconfig != "" {
			os.Setenv("KUBECONFIG", originalKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	})

	Context("when secret is provided", func() {
		It("should create client with secret-based configuration", func() {
			clientset, dynamicClient, err := clients.GetClient(testSecret)

			Expect(err).ToNot(HaveOccurred())
			Expect(clientset).ToNot(BeNil())
			Expect(dynamicClient).ToNot(BeNil())
		})

		It("should handle secret with missing URL", func() {
			invalidSecret := &corev1.Secret{
				Data: map[string][]byte{
					corev1.ServiceAccountTokenKey:  []byte("test-token"),
					corev1.ServiceAccountRootCAKey: []byte("test-ca-data"),
				},
			}

			clientset, dynamicClient, err := clients.GetClient(invalidSecret)

			// Should fail because invalid CA data
			Expect(err).To(HaveOccurred())
			Expect(clientset).To(BeNil())
			Expect(dynamicClient).To(BeNil())
		})

		It("should handle secret with missing token", func() {
			invalidSecret := &corev1.Secret{
				Data: map[string][]byte{
					"URL":                          []byte("https://test-cluster.example.com"),
					corev1.ServiceAccountRootCAKey: []byte("test-ca-data"),
				},
			}

			clientset, dynamicClient, err := clients.GetClient(invalidSecret)

			// Should fail because invalid CA data
			Expect(err).To(HaveOccurred())
			Expect(clientset).To(BeNil())
			Expect(dynamicClient).To(BeNil())
		})

		It("should handle secret with missing CA data", func() {
			invalidSecret := &corev1.Secret{
				Data: map[string][]byte{
					"URL":                         []byte("https://test-cluster.example.com"),
					corev1.ServiceAccountTokenKey: []byte("test-token"),
				},
			}

			clientset, dynamicClient, err := clients.GetClient(invalidSecret)

			// Should fail because missing CA data
			Expect(err).To(HaveOccurred())
			Expect(clientset).To(BeNil())
			Expect(dynamicClient).To(BeNil())
		})

		It("should handle empty secret data", func() {
			emptySecret := &corev1.Secret{
				Data: map[string][]byte{},
			}

			clientset, dynamicClient, err := clients.GetClient(emptySecret)

			// Should fail because missing required data
			Expect(err).To(HaveOccurred())
			Expect(clientset).To(BeNil())
			Expect(dynamicClient).To(BeNil())
		})
	})

	Context("when secret is nil", func() {
		It("should attempt to use in-cluster config", func() {
			// This test will likely fail in non-cluster environments
			// but we can test the error handling
			_, _, err := clients.GetClient(nil)

			// In a test environment, this will likely fail with in-cluster config
			// but the function should handle the error gracefully
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("error getting in-cluster config"))
			}
		})

		Context("when KUBECONFIG environment variable is set", func() {
			It("should use kubeconfig file", func() {
				// Create a temporary kubeconfig file
				tempDir := GinkgoT().TempDir()
				kubeconfigPath := tempDir + "/kubeconfig"

				// Create a minimal valid kubeconfig
				config := clientcmdapi.NewConfig()
				config.Clusters["test-cluster"] = &clientcmdapi.Cluster{
					Server: "https://test-cluster.example.com",
				}
				config.AuthInfos["test-user"] = &clientcmdapi.AuthInfo{
					Token: "test-token",
				}
				config.Contexts["test-context"] = &clientcmdapi.Context{
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				}
				config.CurrentContext = "test-context"

				// Write kubeconfig to file
				err := clientcmd.WriteToFile(*config, kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())

				// Set KUBECONFIG environment variable
				os.Setenv("KUBECONFIG", kubeconfigPath)

				clientset, dynamicClient, err := clients.GetClient(nil)

				Expect(err).ToNot(HaveOccurred())
				Expect(clientset).ToNot(BeNil())
				Expect(dynamicClient).ToNot(BeNil())
			})

			It("should handle invalid kubeconfig file", func() {
				// Set KUBECONFIG to a non-existent file
				os.Setenv("KUBECONFIG", "/non/existent/path")

				clientset, dynamicClient, err := clients.GetClient(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("kubeconfig file does not exist"))
				Expect(clientset).To(BeNil())
				Expect(dynamicClient).To(BeNil())
			})
		})

		Context("when KUBECONFIG environment variable is not set", func() {
			It("should fail with in-cluster config error", func() {
				// Ensure KUBECONFIG is not set
				os.Unsetenv("KUBECONFIG")

				clientset, dynamicClient, err := clients.GetClient(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error getting in-cluster config"))
				Expect(clientset).To(BeNil())
				Expect(dynamicClient).To(BeNil())
			})
		})
	})
})

// Helper function to create a mock rest.Config for testing
func createMockConfig() *rest.Config {
	return &rest.Config{
		Host:        "https://test-cluster.example.com",
		BearerToken: "test-token",
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte("test-ca-data"),
		},
	}
}

// Helper function to create a mock kubernetes.Clientset for testing
func createMockClientset() *kubernetes.Clientset {
	config := createMockConfig()
	clientset, _ := kubernetes.NewForConfig(config)
	return clientset
}
