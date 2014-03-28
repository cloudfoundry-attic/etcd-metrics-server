package health_check_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/gunk/test_server"

	. "github.com/cloudfoundry-incubator/etcd-metrics-server/health_check"
)

var _ = Describe("HealthCheck", func() {
	var (
		server *test_server.Server
		check  *HealthCheck
	)

	Context("when the server is up", func() {
		BeforeEach(func() {
			server = test_server.New()

			check = New(
				server.HTTPTestServer.Listener.Addr().Network(),
				server.HTTPTestServer.Listener.Addr().String(),
				gosteno.NewLogger("health-check-test"),
			)
		})

		AfterEach(func() {
			server.Close()
		})

		It("returns true", func() {
			Ω(check.Ok()).Should(BeTrue())
		})
	})

	Context("when the server is down", func() {
		BeforeEach(func() {
			check = New("tcp", "127.0.0.1:0", gosteno.NewLogger("health-check-test"))
		})

		It("returns false", func() {
			Ω(check.Ok()).Should(BeFalse())
		})
	})
})
