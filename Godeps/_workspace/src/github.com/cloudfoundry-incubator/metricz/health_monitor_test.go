package metricz_test

import (
	. "github.com/cloudfoundry-incubator/metricz"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewDummyHealthMonitor", func() {
	It("creates a health monitor that always reports OK", func() {
		monitor := NewDummyHealthMonitor()
		Ω(monitor.Ok()).Should(BeTrue())
	})
})
