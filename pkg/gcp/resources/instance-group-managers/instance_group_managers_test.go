package instancegroupmanagers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	"k8s.io/utils/ptr"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var _ = Describe("InstanceGroupManagers", func() {
	var (
		ctx    context.Context
		config *utils.Config
		igm    *InstanceGroupManagers
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctx = log.Logger.WithContext(ctx)
		config = &utils.Config{
			Client: &utils.ClientSet{
				// Non-nil stub so New() passes the nil check; not connected to GCP.
				InstanceGroupManagers: &compute.InstanceGroupManagersClient{},
			},
			ProjectID:         "test-project",
			Region:            "europe-west1",
			DefaultPeriodType: "down",
		}
	})

	Describe("New", func() {
		Context("when config is valid", func() {
			BeforeEach(func() {
				config.Period = &period.Period{Type: common.PeriodTypeUp}
			})

			It("should create a new InstanceGroupManagers instance", func() {
				var err error
				igm, err = New(ctx, config)
				Expect(err).NotTo(HaveOccurred())
				Expect(igm).NotTo(BeNil())
				Expect(igm.Config).To(Equal(config))
				Expect(igm.Period).To(Equal(config.Period))
			})
		})

		Context("when config is nil", func() {
			It("should return an error", func() {
				var err error
				igm, err = New(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("config cannot be nil"))
				Expect(igm).To(BeNil())
			})
		})

		Context("when client is nil", func() {
			BeforeEach(func() {
				config.Client = nil
			})

			It("should return an error", func() {
				var err error
				igm, err = New(ctx, config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("GCP client cannot be nil"))
				Expect(igm).To(BeNil())
			})
		})

		Context("when InstanceGroupManagers client is nil", func() {
			BeforeEach(func() {
				config.Client.InstanceGroupManagers = nil
			})

			It("should return an error", func() {
				var err error
				igm, err = New(ctx, config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("GCP InstanceGroupManagers client cannot be nil"))
				Expect(igm).To(BeNil())
			})
		})

		Context("when project ID is empty", func() {
			BeforeEach(func() {
				config.ProjectID = ""
			})

			It("should return an error", func() {
				var err error
				igm, err = New(ctx, config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("project ID cannot be empty"))
				Expect(igm).To(BeNil())
			})
		})
	})

	Describe("getDesiredState", func() {
		BeforeEach(func() {
			igm = &InstanceGroupManagers{Config: config}
		})

		Context("when period is nil", func() {
			BeforeEach(func() {
				igm.Period = nil
			})

			// MIG-stopped instances report "STOPPED", not "TERMINATED" (which is the plain instances.stop value).
			It("should return MIG stopped state", func() {
				state := igm.getDesiredState()
				Expect(state).To(Equal(migInstanceStopped))
			})
		})

		Context("when period type is up", func() {
			BeforeEach(func() {
				igm.Period = &period.Period{Type: common.PeriodTypeUp}
			})

			It("should return running state", func() {
				state := igm.getDesiredState()
				Expect(state).To(Equal(utils.InstanceRunning))
			})
		})

		Context("when period type is down", func() {
			BeforeEach(func() {
				igm.Period = &period.Period{Type: common.PeriodTypeDown}
			})

			It("should return MIG stopped state", func() {
				state := igm.getDesiredState()
				Expect(state).To(Equal(migInstanceStopped))
			})
		})

		Context("when defaultPeriodType is up and period is nil", func() {
			BeforeEach(func() {
				config.DefaultPeriodType = string(common.PeriodTypeUp)
				igm = &InstanceGroupManagers{Config: config}
				igm.Period = nil
			})

			It("should return running state", func() {
				state := igm.getDesiredState()
				Expect(state).To(Equal(utils.InstanceRunning))
			})
		})
	})

	Describe("isMIGSelected", func() {
		BeforeEach(func() {
			igm = &InstanceGroupManagers{Config: config}
		})

		Context("when no names filter is configured", func() {
			It("should select all MIGs", func() {
				mig := &computepb.InstanceGroupManager{Name: ptr.To("any-mig")}
				Expect(igm.isMIGSelected(mig)).To(BeTrue())
			})
		})

		Context("when names filter is configured", func() {
			BeforeEach(func() {
				config.Names = []string{"mig-a", "mig-b"}
			})

			It("should select a MIG whose name matches", func() {
				mig := &computepb.InstanceGroupManager{Name: ptr.To("mig-a")}
				Expect(igm.isMIGSelected(mig)).To(BeTrue())
			})

			It("should not select a MIG whose name does not match", func() {
				mig := &computepb.InstanceGroupManager{Name: ptr.To("mig-c")}
				Expect(igm.isMIGSelected(mig)).To(BeFalse())
			})
		})
	})

	Describe("isManagedInstanceTransitioning", func() {
		It("should return false when CurrentAction is NONE", func() {
			inst := &computepb.ManagedInstance{CurrentAction: ptr.To("NONE")}
			Expect(isManagedInstanceTransitioning(inst)).To(BeFalse())
		})

		It("should return true when CurrentAction is CREATING", func() {
			inst := &computepb.ManagedInstance{CurrentAction: ptr.To("CREATING")}
			Expect(isManagedInstanceTransitioning(inst)).To(BeTrue())
		})

		It("should return true when CurrentAction is RECREATING", func() {
			inst := &computepb.ManagedInstance{CurrentAction: ptr.To("RECREATING")}
			Expect(isManagedInstanceTransitioning(inst)).To(BeTrue())
		})
	})

	Describe("isManagedInstanceInDesiredState", func() {
		It("should return true when instance status matches desired state", func() {
			inst := &computepb.ManagedInstance{InstanceStatus: ptr.To(utils.InstanceRunning)}
			Expect(isManagedInstanceInDesiredState(inst, utils.InstanceRunning)).To(BeTrue())
		})

		It("should return false when instance status does not match desired state", func() {
			inst := &computepb.ManagedInstance{InstanceStatus: ptr.To(utils.InstanceRunning)}
			Expect(isManagedInstanceInDesiredState(inst, migInstanceStopped)).To(BeFalse())
		})

		// Regression: instanceGroupManagers.stopInstances yields "STOPPED" (not "TERMINATED").
		// Using gcpUtils.InstanceStopped ("TERMINATED") would cause every reconcile to re-issue
		// StopInstances against already-stopped instances.
		It("should return true when MIG instance is STOPPED and desired state is STOPPED", func() {
			inst := &computepb.ManagedInstance{InstanceStatus: ptr.To(migInstanceStopped)}
			Expect(isManagedInstanceInDesiredState(inst, migInstanceStopped)).To(BeTrue())
		})
	})

	Describe("applyMIGState", func() {
		BeforeEach(func() {
			igm = &InstanceGroupManagers{Config: config}
		})

		It("should return an error when desired state is unknown", func() {
			err := igm.applyMIGState(ctx, "mig-test", "europe-west1-b", "UNKNOWN", []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown desired state"))
		})
	})
})
