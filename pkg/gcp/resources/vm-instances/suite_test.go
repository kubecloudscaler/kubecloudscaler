package vminstances

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVMnstances(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VMnstances Suite")
}
