package instancegroupmanagers

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstanceGroupManagers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "InstanceGroupManagers Suite")
}
