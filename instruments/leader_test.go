package instruments_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"

	"github.com/cloudfoundry/gunk/test_server"
	"github.com/cloudfoundry/gunk/urljoiner"

	. "github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
	"github.com/cloudfoundry-incubator/metricz/instrumentation"
)

var _ = Describe("Leader Instrumentation", func() {
	var (
		s      *test_server.Server
		leader *Leader
	)

	BeforeEach(func() {
		s = test_server.New()

		leader = NewLeader(s.URL(), lagertest.NewTestLogger("test"))
	})

	Context("when the metrics fetch succesfully", func() {
		AfterEach(func() {
			s.Close()
		})

		Context("when the etcd server is a leader", func() {
			var leaderRequest = test_server.CombineHandlers(
				test_server.VerifyRequest("GET", "/v2/stats/leader"),
				test_server.Respond(200, `
                        {
                          "followers": {
                            "node1": {
                              "counts": {
                                "success": 277031,
                                "fail": 0
                              },
                              "latency": {
                                "maximum": 65.038854,
                                "minimum": 0.124347,
                                "standardDeviation": 0.41350537505117785,
                                "average": 0.37073788356538245,
                                "current": 1.0
                              }
                            },
                            "node2": {
                              "counts": {
                                "success": 277031,
                                "fail": 0
                              },
                              "latency": {
                                "maximum": 65.038854,
                                "minimum": 0.124347,
                                "standardDeviation": 0.41350537505117785,
                                "average": 0.37073788356538245,
                                "current": 2.0
                              }
                            }
                          },
                          "leader": "node0"
                        }
                `),
			)

			BeforeEach(func() {
				s.Append(leaderRequest)
			})

			It("should return them", func() {
				context := leader.Emit()

				Ω(context.Name).Should(Equal("leader"))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "Followers",
					Value: 2,
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "Latency",
					Value: 1.0,
					Tags: map[string]interface{}{
						"follower": "node1",
					},
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "Latency",
					Value: 2.0,
					Tags: map[string]interface{}{
						"follower": "node2",
					},
				}))
			})
		})

		Context("when the etcd server is a follower", func() {
			var leaderRequest = test_server.CombineHandlers(
				test_server.VerifyRequest("GET", "/v2/stats/leader"),
				func(w http.ResponseWriter, req *http.Request) {
					w.Header().Set("Location", urljoiner.Join(s.URL(), "some", "other", "leader"))
					w.WriteHeader(302)
				},
			)

			BeforeEach(func() {
				s.Append(leaderRequest)
			})

			It("does not report any metrics", func() {
				context := leader.Emit()
				Ω(context.Metrics).ShouldNot(BeNil())
				Ω(context.Metrics).Should(BeEmpty())
			})
		})

		Context("when the etcd server gives invalid JSON", func() {
			var leaderRequest = test_server.CombineHandlers(
				test_server.VerifyRequest("GET", "/v2/stats/leader"),
				test_server.Respond(200, "ß"),
			)

			BeforeEach(func() {
				s.Append(leaderRequest)
			})

			It("does not report any metrics", func() {
				context := leader.Emit()
				Ω(context.Metrics).Should(BeEmpty())
			})
		})
	})

	Context("when the metrics fail to fetch", func() {
		BeforeEach(func() {
			s.Close()
		})

		It("should not return them", func() {
			context := leader.Emit()
			Ω(context.Metrics).Should(BeEmpty())
		})
	})
})
