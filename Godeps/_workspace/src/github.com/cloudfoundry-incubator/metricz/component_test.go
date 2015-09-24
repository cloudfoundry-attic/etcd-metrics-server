package metricz_test

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"

	"time"

	. "github.com/cloudfoundry-incubator/metricz"
	"github.com/cloudfoundry-incubator/metricz/instrumentation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("Component", func() {
	var uniquePortForTest uint

	BeforeEach(func() {
		uniquePortForTest = uint(CurrentGinkgoTestDescription().LineNumber + 10000)
	})

	It("component URL", func() {
		component, err := NewComponent(lager.NewLogger("test-component"), "loggregator", uniquePortForTest, GoodHealthMonitor{}, 0, nil, nil)
		Ω(err).ShouldNot(HaveOccurred())

		url := component.URL()

		host, port, err := net.SplitHostPort(url.Host)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(url.Scheme).Should(Equal("http"))

		Ω(host).ShouldNot(Equal("0.0.0.0"))
		Ω(host).ShouldNot(Equal("127.0.0.1"))

		Ω(port).ShouldNot(Equal("0"))
	})

	It("status credentials nil", func() {
		component, err := NewComponent(lager.NewLogger("test-component"), "loggregator", uniquePortForTest, GoodHealthMonitor{}, 0, nil, nil)
		Ω(err).ShouldNot(HaveOccurred())

		url := component.URL()

		Ω(url.User.Username()).ShouldNot(BeEmpty())

		_, passwordPresent := url.User.Password()
		Ω(passwordPresent).Should(BeTrue())
	})

	It("status credentials default", func() {
		component, err := NewComponent(lager.NewLogger("test-component"), "loggregator", uniquePortForTest, GoodHealthMonitor{}, 0, []string{"", ""}, nil)
		Ω(err).ShouldNot(HaveOccurred())

		url := component.URL()

		Ω(url.User.Username()).ShouldNot(BeEmpty())

		_, passwordPresent := url.User.Password()
		Ω(passwordPresent).Should(BeTrue())
	})

	It("good healthz endpoint", func() {
		component, err := NewComponent(
			lager.NewLogger("test-component"),
			"loggregator",
			uniquePortForTest,
			GoodHealthMonitor{},
			0,
			[]string{"user", "pass"},
			[]instrumentation.Instrumentable{},
		)
		Ω(err).ShouldNot(HaveOccurred())

		healthzEndpoint := component.URL().String() + "/healthz"

		go component.StartMonitoringEndpoints()

		req, err := http.NewRequest("GET", healthzEndpoint, nil)
		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(resp.StatusCode, 200)
		Ω(resp.Header.Get("Content-Type"), "text/plain")
		body, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(body)).Should(Equal("ok"))
	})

	It("bad healthz endpoint", func() {
		component, err := NewComponent(
			lager.NewLogger("test-component"),
			"loggregator",
			uniquePortForTest,
			BadHealthMonitor{},
			0,
			[]string{"user", "pass"},
			[]instrumentation.Instrumentable{},
		)
		Ω(err).ShouldNot(HaveOccurred())

		healthzEndpoint := component.URL().String() + "/healthz"
		go component.StartMonitoringEndpoints()

		req, err := http.NewRequest("GET", healthzEndpoint, nil)
		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(resp.StatusCode).Should(Equal(200))
		body, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(body)).Should(Equal("bad"))
	})

	It("panic when failing to monitor endpoints", func() {
		component, err := NewComponent(
			lager.NewLogger("test-component"),
			"loggregator",
			uniquePortForTest,
			GoodHealthMonitor{},
			0,
			[]string{"user", "pass"},
			[]instrumentation.Instrumentable{},
		)
		Ω(err).ShouldNot(HaveOccurred())

		finishChan := make(chan bool)

		go func() {
			defer GinkgoRecover()
			err := component.StartMonitoringEndpoints()
			Ω(err).ShouldNot(HaveOccurred())
		}()
		time.Sleep(50 * time.Millisecond)

		go func() {
			defer GinkgoRecover()

			err := component.StartMonitoringEndpoints()
			Ω(err).Should(HaveOccurred())
			finishChan <- true
		}()

		<-finishChan
	})

	It("stopping server", func() {
		component, err := NewComponent(
			lager.NewLogger("test-component"),
			"loggregator",
			uniquePortForTest,
			GoodHealthMonitor{},
			0,
			[]string{"user", "pass"},
			[]instrumentation.Instrumentable{},
		)
		Ω(err).ShouldNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			err := component.StartMonitoringEndpoints()
			Ω(err).ShouldNot(HaveOccurred())
		}()

		time.Sleep(50 * time.Millisecond)

		component.StopMonitoringEndpoints()

		go func() {
			defer GinkgoRecover()
			err := component.StartMonitoringEndpoints()
			Ω(err).ShouldNot(HaveOccurred())
		}()
	})

	It("varz requires basic auth", func() {
		tags := map[string]interface{}{"tagName1": "tagValue1", "tagName2": "tagValue2"}
		component, err := NewComponent(
			lager.NewLogger("test-component"),
			"loggregator",
			uniquePortForTest,
			GoodHealthMonitor{},
			0,
			[]string{"user", "pass"},
			[]instrumentation.Instrumentable{
				testInstrumentable{
					"agentListener",
					[]instrumentation.Metric{
						instrumentation.Metric{Name: "messagesReceived", Value: 2004},
						instrumentation.Metric{Name: "queueLength", Value: 5, Tags: tags},
					},
				},
				testInstrumentable{
					"cfSinkServer",
					[]instrumentation.Metric{
						instrumentation.Metric{Name: "activeSinkCount", Value: 3},
					},
				},
			},
		)
		Ω(err).ShouldNot(HaveOccurred())

		unauthenticatedURL := component.URL()
		unauthenticatedURL.User = nil
		unauthenticatedURL.Path = "/varz"

		go component.StartMonitoringEndpoints()

		Eventually(func() error {
			conn, err := net.Dial("tcp", unauthenticatedURL.Host)
			if err == nil {
				conn.Close()
			}

			return err
		}).ShouldNot(HaveOccurred())

		req, err := http.NewRequest("GET", unauthenticatedURL.String(), nil)
		Ω(err).ShouldNot(HaveOccurred())
		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp.StatusCode).Should(Equal(401))
	})

	It("varz endpoint", func() {
		tags := map[string]interface{}{"tagName1": "tagValue1", "tagName2": "tagValue2"}
		component, err := NewComponent(
			lager.NewLogger("test-component"),
			"loggregator",
			uniquePortForTest,
			GoodHealthMonitor{},
			0,
			[]string{"user", "pass"},
			[]instrumentation.Instrumentable{
				testInstrumentable{
					"agentListener",
					[]instrumentation.Metric{
						instrumentation.Metric{Name: "messagesReceived", Value: 2004},
						instrumentation.Metric{Name: "queueLength", Value: 5, Tags: tags},
					},
				},
				testInstrumentable{
					"cfSinkServer",
					[]instrumentation.Metric{
						instrumentation.Metric{Name: "activeSinkCount", Value: 3},
					},
				},
			},
		)
		Ω(err).ShouldNot(HaveOccurred())

		varzEndpoint := component.URL().String() + "/varz"

		go component.StartMonitoringEndpoints()

		req, err := http.NewRequest("GET", varzEndpoint, nil)
		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())

		memStats := new(runtime.MemStats)
		runtime.ReadMemStats(memStats)

		Ω(resp.StatusCode).Should(Equal(200))
		Ω(resp.Header.Get("Content-Type")).Should(Equal("application/json"))
		body, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())

		expected := map[string]interface{}{
			"name":          "loggregator",
			"numCPUS":       runtime.NumCPU(),
			"numGoRoutines": runtime.NumGoroutine(),
			"memoryStats": map[string]interface{}{
				"numBytesAllocatedHeap":  int(memStats.HeapAlloc),
				"numBytesAllocatedStack": int(memStats.StackInuse),
				"numBytesAllocated":      int(memStats.Alloc),
				"numMallocs":             int(memStats.Mallocs),
				"numFrees":               int(memStats.Frees),
				"lastGCPauseTimeNS":      int(memStats.PauseNs[(memStats.NumGC+255)%256]),
			},
			"tags": map[string]string{
				"ip": "something",
			},
			"contexts": []interface{}{
				map[string]interface{}{
					"name": "agentListener",
					"metrics": []interface{}{
						map[string]interface{}{
							"name":  "messagesReceived",
							"value": float64(2004),
						},
						map[string]interface{}{
							"name":  "queueLength",
							"value": float64(5),
							"tags": map[string]interface{}{
								"tagName1": "tagValue1",
								"tagName2": "tagValue2",
							},
						},
					},
				},
				map[string]interface{}{
					"name": "cfSinkServer",
					"metrics": []interface{}{
						map[string]interface{}{
							"name":  "activeSinkCount",
							"value": float64(3),
						},
					},
				},
			},
		}

		var actualMap map[string]interface{}
		json.Unmarshal(body, &actualMap)
		Ω(actualMap["tags"]).ShouldNot(BeNil())
		Ω(expected["contexts"]).Should(Equal(actualMap["contexts"]))
		Ω(expected["name"]).Should(Equal(actualMap["name"]))
		Ω(expected["numCPUS"]).Should(BeNumerically("==", actualMap["numCPUS"]))
		Ω(expected["numGoRoutines"]).Should(BeNumerically("==", actualMap["numGoRoutines"]))
		Ω(actualMap["memoryStats"]).ShouldNot(BeNil())
		Ω(actualMap["memoryStats"]).ShouldNot(BeEmpty())
	})
})

type GoodHealthMonitor struct{}

func (hm GoodHealthMonitor) Ok() bool {
	return true
}

type BadHealthMonitor struct{}

func (hm BadHealthMonitor) Ok() bool {
	return false
}

type testInstrumentable struct {
	name    string
	metrics []instrumentation.Metric
}

func (t testInstrumentable) Emit() instrumentation.Context {
	return instrumentation.Context{Name: t.name, Metrics: t.metrics}
}
