package collector_registrar_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCollectorRegistrar(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CollectorRegistrar Suite")
}
