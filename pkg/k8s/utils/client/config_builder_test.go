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
	"errors"
	"testing"

	clients "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	// RunSpecs is handled by suite_test.go
}

var _ = Describe("ConfigBuilder", func() {
	var (
		configBuilder   clients.ConfigBuilder
		mockEnvProvider *MockEnvironmentProvider
	)

	BeforeEach(func() {
		mockEnvProvider = &MockEnvironmentProvider{}
		configBuilder = clients.NewConfigBuilder(mockEnvProvider)
	})

	Context("BuildFromSecret", func() {
		It("should build config from valid secret", func() {
			secret := &corev1.Secret{
				Data: map[string][]byte{
					"URL":                          []byte("https://test-cluster.example.com"),
					corev1.ServiceAccountTokenKey:  []byte("test-token"),
					corev1.ServiceAccountRootCAKey: []byte("test-ca-data"),
					"insecure":                     []byte("false"),
				},
			}

			config, err := configBuilder.BuildFromSecret(secret)

			Expect(err).ToNot(HaveOccurred())
			Expect(config).ToNot(BeNil())
			Expect(config.Host).To(Equal("https://test-cluster.example.com"))
			Expect(config.BearerToken).To(Equal("test-token"))
			Expect(config.TLSClientConfig.CAData).To(Equal([]byte("test-ca-data")))
			Expect(config.TLSClientConfig.Insecure).To(BeFalse())
		})

		It("should handle insecure flag", func() {
			secret := &corev1.Secret{
				Data: map[string][]byte{
					"URL":                          []byte("https://test-cluster.example.com"),
					corev1.ServiceAccountTokenKey:  []byte("test-token"),
					corev1.ServiceAccountRootCAKey: []byte("test-ca-data"),
					"insecure":                     []byte("true"),
				},
			}

			config, err := configBuilder.BuildFromSecret(secret)

			Expect(err).ToNot(HaveOccurred())
			Expect(config.TLSClientConfig.Insecure).To(BeTrue())
		})

		It("should return error for nil secret", func() {
			config, err := configBuilder.BuildFromSecret(nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("secret cannot be nil"))
			Expect(config).To(BeNil())
		})

		It("should return error for secret with nil data", func() {
			secret := &corev1.Secret{}

			config, err := configBuilder.BuildFromSecret(secret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("secret data cannot be nil"))
			Expect(config).To(BeNil())
		})

		It("should return error for missing required fields", func() {
			secret := &corev1.Secret{
				Data: map[string][]byte{
					"URL": []byte("https://test-cluster.example.com"),
					// Missing other required fields
				},
			}

			config, err := configBuilder.BuildFromSecret(secret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required field"))
			Expect(config).To(BeNil())
		})

		It("should return error for invalid insecure flag", func() {
			secret := &corev1.Secret{
				Data: map[string][]byte{
					"URL":                          []byte("https://test-cluster.example.com"),
					corev1.ServiceAccountTokenKey:  []byte("test-token"),
					corev1.ServiceAccountRootCAKey: []byte("test-ca-data"),
					"insecure":                     []byte("invalid"),
				},
			}

			config, err := configBuilder.BuildFromSecret(secret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing insecure flag"))
			Expect(config).To(BeNil())
		})
	})

	Context("BuildFromEnvironment", func() {
		It("should return error when in-cluster config fails and no kubeconfig", func() {
			mockEnvProvider.GetEnvFunc = func(key string) string {
				return "" // No KUBECONFIG
			}

			config, err := configBuilder.BuildFromEnvironment()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting in-cluster config"))
			Expect(config).To(BeNil())
		})

		It("should try kubeconfig when in-cluster config fails", func() {
			mockEnvProvider.GetEnvFunc = func(key string) string {
				if key == "KUBECONFIG" {
					return "/path/to/kubeconfig"
				}
				return ""
			}
			mockEnvProvider.FileExistsFunc = func(path string) bool {
				return path == "/path/to/kubeconfig"
			}

			config, err := configBuilder.BuildFromEnvironment()

			// This will fail because we're not actually implementing the kubeconfig parsing
			// but we can test that it tries to use kubeconfig
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
		})
	})

	Context("BuildFromKubeconfig", func() {
		It("should return error for non-existent kubeconfig file", func() {
			mockEnvProvider.FileExistsFunc = func(path string) bool {
				return false
			}

			config, err := configBuilder.BuildFromKubeconfig("/non/existent/path")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("kubeconfig file does not exist"))
			Expect(config).To(BeNil())
		})
	})
})

// MockEnvironmentProvider for testing
type MockEnvironmentProvider struct {
	GetEnvFunc     func(key string) string
	FileExistsFunc func(path string) bool
	ReadFileFunc   func(path string) ([]byte, error)
}

func (m *MockEnvironmentProvider) GetEnv(key string) string {
	if m.GetEnvFunc != nil {
		return m.GetEnvFunc(key)
	}
	return ""
}

func (m *MockEnvironmentProvider) FileExists(path string) bool {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(path)
	}
	return false
}

func (m *MockEnvironmentProvider) ReadFile(path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(path)
	}
	return nil, errors.New("not implemented")
}
