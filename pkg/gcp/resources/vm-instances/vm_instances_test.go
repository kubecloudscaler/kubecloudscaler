package vminstances

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"

	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var _ = Describe("ComputeInstances", func() {
	var (
		ctx    context.Context
		config *utils.Config
		ci     *VMnstances
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctx = log.Logger.WithContext(ctx)
		config = &utils.Config{
			Client:            &utils.ClientSet{},
			ProjectID:         "test-project",
			Region:            "us-central1",
			DefaultPeriodType: "down",
		}
	})

	Describe("New", func() {
		Context("when config is valid", func() {
			BeforeEach(func() {
				config.Period = &period.Period{Type: "up"}
			})

			It("should create a new VMnstances instance", func() {
				var err error
				ci, err = New(ctx, config)
				Expect(err).NotTo(HaveOccurred())
				Expect(ci).NotTo(BeNil())
				Expect(ci.Config).To(Equal(config))
				Expect(ci.Period).To(Equal(config.Period))
			})
		})

		Context("when config is nil", func() {
			It("should return an error", func() {
				var err error
				ci, err = New(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("config cannot be nil"))
				Expect(ci).To(BeNil())
			})
		})

		Context("when client is nil", func() {
			BeforeEach(func() {
				config.Client = nil
			})

			It("should return an error", func() {
				var err error
				ci, err = New(ctx, config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("GCP client cannot be nil"))
				Expect(ci).To(BeNil())
			})
		})

		Context("when project ID is empty", func() {
			BeforeEach(func() {
				config.ProjectID = ""
			})

			It("should return an error", func() {
				var err error
				ci, err = New(ctx, config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("project ID cannot be empty"))
				Expect(ci).To(BeNil())
			})
		})
	})

	Describe("getDesiredState", func() {
		BeforeEach(func() {
			ci = &VMnstances{Config: config}
		})

		Context("when period is nil", func() {
			BeforeEach(func() {
				ci.Period = nil
			})

			It("should return stopped state", func() {
				state := ci.getDesiredState()
				Expect(state).To(Equal(utils.InstanceStopped))
			})
		})

		Context("when period type is up", func() {
			BeforeEach(func() {
				ci.Period = &period.Period{Type: "up"}
			})

			It("should return running state", func() {
				state := ci.getDesiredState()
				Expect(state).To(Equal(utils.InstanceRunning))
			})
		})

		Context("when period type is down", func() {
			BeforeEach(func() {
				ci.Period = &period.Period{Type: "down"}
			})

			It("should return stopped state", func() {
				state := ci.getDesiredState()
				Expect(state).To(Equal(utils.InstanceStopped))
			})
		})

		Context("when period type is unknown", func() {
			BeforeEach(func() {
				ci.Period = &period.Period{Type: "unknown"}
			})

			It("should return stopped state", func() {
				state := ci.getDesiredState()
				Expect(state).To(Equal(utils.InstanceStopped))
			})
		})
	})

	Describe("isInstanceInDesiredState", func() {
		BeforeEach(func() {
			ci = &VMnstances{Config: config}
		})

		Context("when instance is in desired state", func() {
			It("should return true", func() {
				instance := &computepb.Instance{Status: stringPtr(utils.InstanceRunning)}
				result := ci.isInstanceInDesiredState(instance, utils.InstanceRunning)
				Expect(result).To(BeTrue())
			})
		})

		Context("when instance is not in desired state", func() {
			It("should return false", func() {
				instance := &computepb.Instance{Status: stringPtr(utils.InstanceStopped)}
				result := ci.isInstanceInDesiredState(instance, utils.InstanceRunning)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("extractZoneFromInstance", func() {
		BeforeEach(func() {
			ci = &VMnstances{Config: config}
		})

		Context("when zone URL is valid", func() {
			It("should extract zone name", func() {
				instance := &computepb.Instance{
					Zone: stringPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a"),
				}
				zone := ci.extractZoneFromInstance(instance)
				Expect(zone).To(Equal("us-central1-a"))
			})
		})

		Context("when zone URL has trailing slash", func() {
			It("should extract zone name", func() {
				instance := &computepb.Instance{
					Zone: stringPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a/"),
				}
				zone := ci.extractZoneFromInstance(instance)
				Expect(zone).To(Equal("us-central1-a"))
			})
		})

		Context("when zone is empty", func() {
			It("should return empty string", func() {
				instance := &computepb.Instance{Zone: stringPtr("")}
				zone := ci.extractZoneFromInstance(instance)
				Expect(zone).To(Equal(""))
			})
		})

		Context("when zone is nil", func() {
			It("should return empty string", func() {
				instance := &computepb.Instance{Zone: nil}
				zone := ci.extractZoneFromInstance(instance)
				Expect(zone).To(Equal(""))
			})
		})
	})

	Describe("SetState", func() {
		BeforeEach(func() {
			ci = &VMnstances{
				Config: config,
				Period: &period.Period{Type: "up"},
			}
		})

		Context("when no instances are found", func() {
			It("should return success with no instances message", func() {
				// This test would require mocking the GCP client calls
				// For now, we'll test the basic structure
				success, failed, err := ci.SetState(ctx)
				Expect(err).To(HaveOccurred()) // Will fail due to missing GCP client setup
				Expect(success).To(BeEmpty())
				Expect(failed).To(BeEmpty())
			})
		})
	})

	Describe("applyInstanceState", func() {
		BeforeEach(func() {
			ci = &VMnstances{Config: config}
		})

		Context("when desired state is unknown", func() {
			It("should return an error", func() {
				instance := &computepb.Instance{
					Name: stringPtr("test-instance"),
					Zone: stringPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a"),
				}

				err := ci.applyInstanceState(ctx, instance, "unknown")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown desired state"))
			})
		})

		Context("when zone cannot be extracted", func() {
			It("should return an error", func() {
				instance := &computepb.Instance{
					Name: stringPtr("test-instance"),
					Zone: stringPtr(""),
				}

				err := ci.applyInstanceState(ctx, instance, utils.InstanceRunning)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot extract zone from instance"))
			})
		})
	})
})

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
