package pack

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pack Suite")
}
