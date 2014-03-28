package instruments_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestInstruments(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instruments Suite")
}
