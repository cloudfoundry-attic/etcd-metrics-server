package health_check_test

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/cloudfoundry-incubator/etcd-metrics-server/health_check"
)

var _ = Describe("HealthCheck", func() {
	var (
		server *ghttp.Server
		check  *HealthCheck
	)

	Context("when the server is up", func() {
		BeforeEach(func() {
			server = ghttp.NewServer()

			check = New(
				server.HTTPTestServer.Listener.Addr().Network(),
				server.HTTPTestServer.Listener.Addr().String(),
				lagertest.NewTestLogger("test"),
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
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			Ω(err).ShouldNot(HaveOccurred())

			err = listener.Close()
			Ω(err).ShouldNot(HaveOccurred())

			check = New("tcp", listener.Addr().String(), lagertest.NewTestLogger("test"))
		})

		It("returns false", func() {
			Ω(check.Ok()).Should(BeFalse())
		})
	})
})
