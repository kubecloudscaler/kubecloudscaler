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

package utils

import (
	"fmt"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// TestEnvironment holds the test environment and client for controller tests.
type TestEnvironment struct {
	Config    *rest.Config
	Client    client.Client
	TestEnv   *envtest.Environment
	CRDPaths  []string
	SuiteName string
}

// SetupTestEnvironment sets up a test environment for controller tests.
// This consolidates the common test setup logic used by both k8s and gcp controllers.
//
// Parameters:
//   - crdPaths: Paths to CRD directories (relative to project root)
//   - suiteName: Name of the test suite
//
// Returns:
//   - *TestEnvironment: The configured test environment
func SetupTestEnvironment(crdPaths []string, suiteName string) *TestEnvironment {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     crdPaths,
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kubecloudscalerv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	return &TestEnvironment{
		Config:    cfg,
		Client:    k8sClient,
		TestEnv:   testEnv,
		CRDPaths:  crdPaths,
		SuiteName: suiteName,
	}
}

// TeardownTestEnvironment tears down the test environment.
func (te *TestEnvironment) TeardownTestEnvironment() {
	By("tearing down the test environment")
	err := te.TestEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
}
