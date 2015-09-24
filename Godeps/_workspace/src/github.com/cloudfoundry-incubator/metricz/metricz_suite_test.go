package metricz_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMetricz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricz Suite")
}
