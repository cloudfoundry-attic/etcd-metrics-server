package instruments_test

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/gunk/test_server"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"

	. "github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
)

var _ = Describe("Store Instrumentation", func() {
	var (
		s     *test_server.Server
		store *Store
	)

	BeforeEach(func() {
		s = test_server.New()
		store = NewStore(s.URL(), gosteno.NewLogger("store-test"))
	})

	Context("when the metrics fetch succesfully", func() {
		AfterEach(func() {
			s.Close()
		})

		stats := map[string]uint64{
			"compareAndSwapFail":    1,
			"compareAndSwapSuccess": 2,
			"createFail":            3,
			"createSuccess":         4,
			"deleteFail":            5,
			"deleteSuccess":         6,
			"expireCount":           7,
			"getsFail":              8,
			"getsSuccess":           9,
			"setsFail":              10,
			"setsSuccess":           11,
			"updateFail":            12,
			"updateSuccess":         13,
			"watchers":              14,
		}

		statsPayload, err := json.Marshal(stats)
		if err != nil {
			panic(err)
		}

		var statsRequest = test_server.CombineHandlers(
			test_server.VerifyRequest("GET", "/v2/stats/store"),
			test_server.Respond(200, string(statsPayload)),
		)

		var keysRequest = test_server.CombineHandlers(
			test_server.VerifyRequest("GET", "/v2/keys/"),
			func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("X-Etcd-Index", "10001")
				w.Header().Set("X-Raft-Index", "10204")
				w.Header().Set("X-Raft-Term", "1234")
				w.WriteHeader(200)
			},
		)

		Context("when the etcd server gives valid JSON", func() {
			BeforeEach(func() {
				s.Append(statsRequest, keysRequest)
			})

			It("should return them", func() {
				context := store.Emit()

				Ω(context.Name).Should(Equal("store"))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "EtcdIndex",
					Value: uint64(10001),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "RaftIndex",
					Value: uint64(10204),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "RaftTerm",
					Value: uint64(1234),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "CompareAndSwapFail",
					Value: uint64(1),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "CompareAndSwapSuccess",
					Value: uint64(2),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "CreateFail",
					Value: uint64(3),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "CreateSuccess",
					Value: uint64(4),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "DeleteFail",
					Value: uint64(5),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "DeleteSuccess",
					Value: uint64(6),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "ExpireCount",
					Value: uint64(7),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "GetsFail",
					Value: uint64(8),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "GetsSuccess",
					Value: uint64(9),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "SetsFail",
					Value: uint64(10),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "SetsSuccess",
					Value: uint64(11),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "UpdateFail",
					Value: uint64(12),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "UpdateSuccess",
					Value: uint64(13),
				}))

				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{
					Name:  "Watchers",
					Value: uint64(14),
				}))
			})
		})

		Context("when the etcd server gives invalid JSON", func() {
			var statsRequest = test_server.CombineHandlers(
				test_server.VerifyRequest("GET", "/v2/stats/store"),
				test_server.Respond(200, "ß"),
			)

			BeforeEach(func() {
				s.Append(statsRequest)
			})

			It("does not report any metrics", func() {
				context := store.Emit()
				Ω(context.Metrics).Should(BeEmpty())
			})
		})

		Context("when getting the keys fails", func() {
			var keysRequest = test_server.CombineHandlers(
				test_server.VerifyRequest("GET", "/v2/keys/"),
				test_server.Respond(404, ""),
			)

			BeforeEach(func() {
				s.Append(statsRequest, keysRequest)
			})

			It("does not report any metrics", func() {
				context := store.Emit()
				Ω(context.Metrics).Should(BeEmpty())
			})
		})
	})

	Context("when the metrics fail to fetch", func() {
		BeforeEach(func() {
			s.Close()
		})

		It("should not return them", func() {
			context := store.Emit()
			Ω(context.Metrics).Should(BeEmpty())
		})
	})
})
