package utils

import (
	"context"

	computepb "cloud.google.com/go/compute/apiv1/computepb"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GCP Utils", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("GetZonesFromRegion", func() {
		Context("when clients is nil", func() {
			It("should return an error", func() {
				zones, err := GetZonesFromRegion(ctx, nil, "test-project", "us-central1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("regions client is nil"))
				Expect(zones).To(BeNil())
			})
		})

		Context("when regions client is nil", func() {
			It("should return an error", func() {
				zones, err := GetZonesFromRegion(ctx, &ClientSet{}, "test-project", "us-central1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("regions client is nil"))
				Expect(zones).To(BeNil())
			})
		})
	})

	Describe("GetInstancesInZones", func() {
		Context("when clients is nil", func() {
			It("should return an error", func() {
				instances, err := GetInstancesInZones(ctx, nil, "test-project", []string{"us-central1-a"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("instances client is nil"))
				Expect(instances).To(BeNil())
			})
		})

		Context("when instances client is nil", func() {
			It("should return an error", func() {
				instances, err := GetInstancesInZones(ctx, &ClientSet{}, "test-project", []string{"us-central1-a"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("instances client is nil"))
				Expect(instances).To(BeNil())
			})
		})
	})

	Describe("FilterInstancesByLabels", func() {
		var (
			instances []*computepb.Instance
		)

		BeforeEach(func() {
			instances = []*computepb.Instance{
				{
					Name:   stringPtr("instance-1"),
					Labels: map[string]string{"env": "prod", "app": "web"},
				},
				{
					Name:   stringPtr("instance-2"),
					Labels: map[string]string{"env": "dev", "app": "api"},
				},
				{
					Name:   stringPtr("instance-3"),
					Labels: map[string]string{"env": "prod", "app": "api"},
				},
				{
					Name:   stringPtr("instance-4"),
					Labels: nil,
				},
			}
		})

		Context("when label selector is nil", func() {
			It("should return all instances", func() {
				filtered := FilterInstancesByLabels(instances, nil)
				Expect(filtered).To(HaveLen(4))
				Expect(filtered).To(Equal(instances))
			})
		})

		Context("when filtering by env=prod", func() {
			It("should return only prod instances", func() {
				selector := &metaV1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}
				filtered := FilterInstancesByLabels(instances, selector)
				Expect(filtered).To(HaveLen(2))
				Expect(filtered[0].GetName()).To(Equal("instance-1"))
				Expect(filtered[1].GetName()).To(Equal("instance-3"))
			})
		})

		Context("when filtering by app=api", func() {
			It("should return only api instances", func() {
				selector := &metaV1.LabelSelector{
					MatchLabels: map[string]string{"app": "api"},
				}
				filtered := FilterInstancesByLabels(instances, selector)
				Expect(filtered).To(HaveLen(2))
				Expect(filtered[0].GetName()).To(Equal("instance-2"))
				Expect(filtered[1].GetName()).To(Equal("instance-3"))
			})
		})

		Context("when filtering by multiple labels", func() {
			It("should return instances matching all labels", func() {
				selector := &metaV1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod", "app": "web"},
				}
				filtered := FilterInstancesByLabels(instances, selector)
				Expect(filtered).To(HaveLen(1))
				Expect(filtered[0].GetName()).To(Equal("instance-1"))
			})
		})

		Context("when filtering by non-existent label", func() {
			It("should return no instances", func() {
				selector := &metaV1.LabelSelector{
					MatchLabels: map[string]string{"nonexistent": "value"},
				}
				filtered := FilterInstancesByLabels(instances, selector)
				Expect(filtered).To(BeEmpty())
			})
		})
	})

	Describe("GetInstanceStatus", func() {
		Context("when instance has status", func() {
			It("should return the status", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceRunning)}
				status := GetInstanceStatus(instance)
				Expect(status).To(Equal(InstanceRunning))
			})
		})

		Context("when instance status is nil", func() {
			It("should return empty string", func() {
				instance := &computepb.Instance{Status: nil}
				status := GetInstanceStatus(instance)
				Expect(status).To(Equal(""))
			})
		})
	})

	Describe("IsInstanceRunning", func() {
		Context("when instance is running", func() {
			It("should return true", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceRunning)}
				result := IsInstanceRunning(instance)
				Expect(result).To(BeTrue())
			})
		})

		Context("when instance is not running", func() {
			It("should return false", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceStopped)}
				result := IsInstanceRunning(instance)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("IsInstanceStopped", func() {
		Context("when instance is stopped", func() {
			It("should return true", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceStopped)}
				result := IsInstanceStopped(instance)
				Expect(result).To(BeTrue())
			})
		})

		Context("when instance is not stopped", func() {
			It("should return false", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceRunning)}
				result := IsInstanceStopped(instance)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("IsInstanceTransitioning", func() {
		Context("when instance is starting", func() {
			It("should return true", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceStarting)}
				result := IsInstanceTransitioning(instance)
				Expect(result).To(BeTrue())
			})
		})

		Context("when instance is stopping", func() {
			It("should return true", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceStopping)}
				result := IsInstanceTransitioning(instance)
				Expect(result).To(BeTrue())
			})
		})

		Context("when instance is running", func() {
			It("should return false", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceRunning)}
				result := IsInstanceTransitioning(instance)
				Expect(result).To(BeFalse())
			})
		})

		Context("when instance is stopped", func() {
			It("should return false", func() {
				instance := &computepb.Instance{Status: stringPtr(InstanceStopped)}
				result := IsInstanceTransitioning(instance)
				Expect(result).To(BeFalse())
			})
		})
	})
})

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
