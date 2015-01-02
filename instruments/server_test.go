package instruments_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
	"github.com/cloudfoundry-incubator/metricz/instrumentation"
)

var _ = Describe("Server Instrumentation", func() {
	var (
		s      *ghttp.Server
		server *Server
	)

	BeforeEach(func() {
		s = ghttp.NewServer()
		server = NewServer(s.URL(), lagertest.NewTestLogger("test"))
	})

	Context("when the metrics fetch succesfully", func() {
		AfterEach(func() {
			s.Close()
		})

		Context("when the etcd server gives valid JSON", func() {
			var leaderRequest = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/stats/self"),
				ghttp.RespondWith(200, `
                    {
                        "name": "node1",
                        "state": "leader",

                        "leaderInfo": {
                            "name": "node1",
                            "uptime": "forever"
                        },

                        "recvAppendRequestCnt": 1234,
                        "recvPkgRate": 5678.0,
                        "recvBandwidthRate": 9101112.13,

                        "sendAppendRequestCnt": 4321,
                        "sendPkgRate": 8765.0,
                        "sendBandwidthRate": 1211109.8
                    }
                `),
			)

			BeforeEach(func() {
				s.AppendHandlers(leaderRequest)
			})

			It("should return them", func() {
				context := server.Emit()

				Ω(context.Name).Should(Equal("server"))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "IsLeader",
					Value: 1,
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "SendingBandwidthRate",
					Value: 1211109.8,
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "ReceivingBandwidthRate",
					Value: 9101112.13,
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "SendingRequestRate",
					Value: 8765.0,
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "ReceivingRequestRate",
					Value: 5678.0,
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "SentAppendRequests",
					Value: uint64(4321),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "ReceivedAppendRequests",
					Value: uint64(1234),
				}))
			})
		})

		Context("when the etcd server gives invalid JSON", func() {
			var leaderRequest = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/stats/self"),
				ghttp.RespondWith(200, "ß"),
			)

			BeforeEach(func() {
				s.AppendHandlers(leaderRequest)
			})

			It("does not report any metrics", func() {
				context := server.Emit()
				Ω(context.Metrics).Should(BeEmpty())
			})
		})
	})

	Context("when the metrics fail to fetch", func() {
		BeforeEach(func() {
			s.Close()
		})

		It("should not return them", func() {
			context := server.Emit()
			Ω(context.Metrics).Should(BeEmpty())
		})
	})
})
