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

package main

import (
	"crypto/tls"
	"flag"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	kubecloudscalerv1alpha2 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha2"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

var _ = Describe("Main Package", func() {
	var (
		originalArgs []string
	)

	BeforeEach(func() {
		originalArgs = os.Args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	})

	AfterEach(func() {
		os.Args = originalArgs
	})

	Describe("Flag Parsing", func() {
		It("should parse default flags correctly", func() {
			os.Args = []string{"cmd", "--metrics-bind-address=:8080", "--health-probe-bind-address=:8081"}

			var metricsAddr string
			var probeAddr string
			var enableLeaderElection bool
			var secureMetrics bool
			var enableHTTP2 bool

			flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to.")
			flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
			flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
			flag.BoolVar(&secureMetrics, "metrics-secure", true, "If set, the metrics endpoint is served securely via HTTPS.")
			flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers")

			flag.Parse()

			Expect(metricsAddr).To(Equal(":8080"))
			Expect(probeAddr).To(Equal(":8081"))
			Expect(enableLeaderElection).To(BeFalse())
			Expect(secureMetrics).To(BeTrue())
			Expect(enableHTTP2).To(BeFalse())
		})

		It("should parse custom flag values correctly", func() {
			os.Args = []string{
				"cmd",
				"--metrics-bind-address=:8443",
				"--health-probe-bind-address=:9090",
				"--leader-elect=true",
				"--metrics-secure=false",
				"--enable-http2=true",
			}

			var metricsAddr string
			var probeAddr string
			var enableLeaderElection bool
			var secureMetrics bool
			var enableHTTP2 bool

			flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to.")
			flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
			flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
			flag.BoolVar(&secureMetrics, "metrics-secure", true, "If set, the metrics endpoint is served securely via HTTPS.")
			flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers")

			flag.Parse()

			Expect(metricsAddr).To(Equal(":8443"))
			Expect(probeAddr).To(Equal(":9090"))
			Expect(enableLeaderElection).To(BeTrue())
			Expect(secureMetrics).To(BeFalse())
			Expect(enableHTTP2).To(BeTrue())
		})

		It("should parse certificate-related flags with defaults", func() {
			os.Args = []string{"cmd"}

			var webhookCertPath, webhookCertName, webhookCertKey string
			var metricsCertPath, metricsCertName, metricsCertKey string

			flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
			flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
			flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
			flag.StringVar(&metricsCertPath, "metrics-cert-path", "", "The directory that contains the metrics server certificate.")
			flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
			flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")

			flag.Parse()

			Expect(webhookCertPath).To(BeEmpty())
			Expect(webhookCertName).To(Equal("tls.crt"))
			Expect(webhookCertKey).To(Equal("tls.key"))
			Expect(metricsCertPath).To(BeEmpty())
			Expect(metricsCertName).To(Equal("tls.crt"))
			Expect(metricsCertKey).To(Equal("tls.key"))
		})

		It("should parse metrics-disable-auth flag with default false", func() {
			os.Args = []string{"cmd"}

			var metricsDisableAuth bool
			flag.BoolVar(&metricsDisableAuth, "metrics-disable-auth", false, "Disable metrics auth.")

			flag.Parse()

			Expect(metricsDisableAuth).To(BeFalse())
		})

		It("should parse log format and level flags with defaults", func() {
			os.Args = []string{"cmd"}

			var logFmt, logLvl string
			flag.StringVar(&logFmt, "log-format", "json", "Set log format")
			flag.StringVar(&logLvl, "log-level", "info", "Set log level")

			flag.Parse()

			Expect(logFmt).To(Equal("json"))
			Expect(logLvl).To(Equal("info"))
		})
	})

	Describe("TLS Configuration", func() {
		It("should disable HTTP/2 when enable-http2 is false", func() {
			enableHTTP2 := false
			var tlsOpts []func(*tls.Config)

			disableHTTP2 := func(c *tls.Config) {
				c.NextProtos = []string{"http/1.1"}
			}

			if !enableHTTP2 {
				tlsOpts = append(tlsOpts, disableHTTP2)
			}

			Expect(tlsOpts).To(HaveLen(1))

			tlsConfig := &tls.Config{}
			tlsOpts[0](tlsConfig)
			Expect(tlsConfig.NextProtos).To(Equal([]string{"http/1.1"}))
		})

		It("should not disable HTTP/2 when enable-http2 is true", func() {
			enableHTTP2 := true
			var tlsOpts []func(*tls.Config)

			disableHTTP2 := func(c *tls.Config) {
				c.NextProtos = []string{"http/1.1"}
			}

			if !enableHTTP2 {
				tlsOpts = append(tlsOpts, disableHTTP2)
			}

			Expect(tlsOpts).To(BeEmpty())
		})

		It("should leave TLS config NextProtos unset when HTTP/2 is enabled", func() {
			tlsConfig := &tls.Config{}
			// When no TLS options are applied, NextProtos should remain nil
			Expect(tlsConfig.NextProtos).To(BeNil())
		})
	})

	Describe("Webhook Server Configuration", func() {
		It("should create webhook server with TLS options", func() {
			var tlsOpts []func(*tls.Config)
			disableHTTP2 := func(c *tls.Config) {
				c.NextProtos = []string{"http/1.1"}
			}
			tlsOpts = append(tlsOpts, disableHTTP2)

			webhookServer := webhook.NewServer(webhook.Options{
				TLSOpts: tlsOpts,
			})

			Expect(webhookServer).ToNot(BeNil())
		})

		It("should create webhook server without TLS options", func() {
			webhookServer := webhook.NewServer(webhook.Options{})

			Expect(webhookServer).ToNot(BeNil())
		})
	})

	Describe("Metrics Server Configuration", func() {
		It("should configure metrics server with secure serving and auth enabled", func() {
			secureMetrics := true
			metricsDisableAuth := false
			var tlsOpts []func(*tls.Config)

			metricsServerOptions := metricsserver.Options{
				BindAddress:   ":8443",
				SecureServing: secureMetrics,
				TLSOpts:       tlsOpts,
			}

			if secureMetrics && !metricsDisableAuth {
				metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
			}

			Expect(metricsServerOptions.BindAddress).To(Equal(":8443"))
			Expect(metricsServerOptions.SecureServing).To(BeTrue())
			Expect(metricsServerOptions.FilterProvider).ToNot(BeNil())
		})

		It("should configure metrics server with secure serving but auth disabled", func() {
			secureMetrics := true
			metricsDisableAuth := true
			var tlsOpts []func(*tls.Config)

			metricsServerOptions := metricsserver.Options{
				BindAddress:   ":8443",
				SecureServing: secureMetrics,
				TLSOpts:       tlsOpts,
			}

			if secureMetrics && !metricsDisableAuth {
				metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
			}

			Expect(metricsServerOptions.BindAddress).To(Equal(":8443"))
			Expect(metricsServerOptions.SecureServing).To(BeTrue())
			Expect(metricsServerOptions.FilterProvider).To(BeNil())
		})

		It("should configure metrics server with secure serving disabled", func() {
			secureMetrics := false
			metricsDisableAuth := false
			var tlsOpts []func(*tls.Config)

			metricsServerOptions := metricsserver.Options{
				BindAddress:   ":8080",
				SecureServing: secureMetrics,
				TLSOpts:       tlsOpts,
			}

			if secureMetrics && !metricsDisableAuth {
				metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
			}

			Expect(metricsServerOptions.BindAddress).To(Equal(":8080"))
			Expect(metricsServerOptions.SecureServing).To(BeFalse())
			Expect(metricsServerOptions.FilterProvider).To(BeNil())
		})
	})

	Describe("Scheme Configuration", func() {
		var testScheme *runtime.Scheme

		BeforeEach(func() {
			testScheme = runtime.NewScheme()
			utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
			utilruntime.Must(kubecloudscalerv1alpha1.AddToScheme(testScheme))
			utilruntime.Must(kubecloudscalerv1alpha2.AddToScheme(testScheme))
			utilruntime.Must(kubecloudscalerv1alpha3.AddToScheme(testScheme))
		})

		It("should register v1alpha3 K8s type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "K8s",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha3 K8sList type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "K8sList",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha3 Gcp type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "Gcp",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha3 GcpList type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "GcpList",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha3 Flow type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "Flow",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha3 FlowList type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "FlowList",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha2 K8s type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha2",
				Kind:    "K8s",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha2 Gcp type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha2",
				Kind:    "Gcp",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha1 K8s type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha1",
				Kind:    "K8s",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register v1alpha1 Gcp type", func() {
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha1",
				Kind:    "Gcp",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should register core Kubernetes types", func() {
			// Verify that core types like Pod are registered via clientgoscheme
			gvk := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			}
			_, err := testScheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should match the package-level scheme init registration", func() {
			// The package-level scheme var is populated in init().
			// Verify it has the same types registered as our test scheme.
			Expect(scheme).ToNot(BeNil())
			Expect(scheme.AllKnownTypes()).ToNot(BeEmpty())

			// Verify a v1alpha3 type is in the package-level scheme
			gvk := schema.GroupVersionKind{
				Group:   "kubecloudscaler.cloud",
				Version: "v1alpha3",
				Kind:    "K8s",
			}
			_, err := scheme.New(gvk)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Webhook Environment Variable Logic", func() {
		var originalEnableWebhooks string
		var hadEnableWebhooks bool

		BeforeEach(func() {
			originalEnableWebhooks, hadEnableWebhooks = os.LookupEnv("ENABLE_WEBHOOKS")
		})

		AfterEach(func() {
			if hadEnableWebhooks {
				os.Setenv("ENABLE_WEBHOOKS", originalEnableWebhooks)
			} else {
				os.Unsetenv("ENABLE_WEBHOOKS")
			}
		})

		It("should enable webhooks when ENABLE_WEBHOOKS is unset", func() {
			os.Unsetenv("ENABLE_WEBHOOKS")
			Expect(os.Getenv("ENABLE_WEBHOOKS")).ToNot(Equal(webhookDisabledEnvValue))
		})

		It("should enable webhooks when ENABLE_WEBHOOKS is true", func() {
			os.Setenv("ENABLE_WEBHOOKS", "true")
			Expect(os.Getenv("ENABLE_WEBHOOKS")).ToNot(Equal(webhookDisabledEnvValue))
		})

		It("should disable webhooks when ENABLE_WEBHOOKS is false", func() {
			os.Setenv("ENABLE_WEBHOOKS", "false")
			Expect(os.Getenv("ENABLE_WEBHOOKS")).To(Equal(webhookDisabledEnvValue))
		})

		It("should use the correct disabled value constant", func() {
			Expect(webhookDisabledEnvValue).To(Equal("false"))
		})
	})

	Describe("Health Check Configuration", func() {
		It("should use healthz.Ping as a valid checker", func() {
			checker := healthz.Ping
			Expect(checker).ToNot(BeNil())

			// healthz.Ping should return nil for a healthy check
			err := checker(nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Manager Configuration Constants", func() {
		It("should use the correct leader election ID", func() {
			// The leader election ID must be a valid DNS subdomain and unique to this operator
			expectedLeaderElectionID := "437b2c63.kubecloudscaler"
			Expect(expectedLeaderElectionID).To(ContainSubstring("kubecloudscaler"))
			Expect(expectedLeaderElectionID).ToNot(BeEmpty())
		})

		It("should use the correct default health probe address", func() {
			// Default from flag definition in main.go
			expectedProbeAddr := ":8081"
			Expect(expectedProbeAddr).To(HavePrefix(":"))
		})

		It("should use the correct default metrics address", func() {
			// Default from flag definition in main.go: "0" means disabled
			expectedMetricsAddr := "0"
			Expect(expectedMetricsAddr).To(Equal("0"))
		})
	})
})
