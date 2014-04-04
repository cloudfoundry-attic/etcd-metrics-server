package metricz

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"github.com/cloudfoundry/loggregatorlib/loggertesthelper"
	"github.com/stretchr/testify/assert"
)

type GoodHealthMonitor struct{}

func (hm GoodHealthMonitor) Ok() bool {
	return true
}

type BadHealthMonitor struct{}

func (hm BadHealthMonitor) Ok() bool {
	return false
}

func TestIpAddressDefault(t *testing.T) {
	component, err := NewComponent(loggertesthelper.Logger(), "loggregator", 0, GoodHealthMonitor{}, 0, nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, component.IpAddress)
	assert.NotEqual(t, "0.0.0.0", component.IpAddress)
	assert.NotEqual(t, "127.0.0.1", component.IpAddress)
}

func TestStatusPortDefault(t *testing.T) {
	component, err := NewComponent(loggertesthelper.Logger(), "loggregator", 0, GoodHealthMonitor{}, 0, nil, nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uint32(0), component.StatusPort)
}

func TestStatusCredentialsNil(t *testing.T) {
	component, err := NewComponent(loggertesthelper.Logger(), "loggregator", 0, GoodHealthMonitor{}, 0, nil, nil)
	assert.NoError(t, err)
	credentials := component.StatusCredentials
	assert.Equal(t, 2, len(credentials))
	assert.NotEmpty(t, credentials[0])
	assert.NotEmpty(t, credentials[1])
}

func TestStatusCredentialsDefault(t *testing.T) {
	component, err := NewComponent(loggertesthelper.Logger(), "loggregator", 0, GoodHealthMonitor{}, 0, []string{"", ""}, nil)
	assert.NoError(t, err)
	credentials := component.StatusCredentials
	assert.Equal(t, 2, len(credentials))
	assert.NotEmpty(t, credentials[0])
	assert.NotEmpty(t, credentials[1])
}

func TestGoodHealthzEndpoint(t *testing.T) {
	component := &Component{
		Logger:            loggertesthelper.Logger(),
		HealthMonitor:     GoodHealthMonitor{},
		StatusPort:        7877,
		Type:              "loggregator",
		StatusCredentials: []string{"user", "pass"},
	}

	go component.StartMonitoringEndpoints()

	req, err := http.NewRequest("GET", "http://localhost:7877/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "text/plain")
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, string(body), "ok")
}

func TestBadHealthzEndpoint(t *testing.T) {
	component := &Component{
		Logger:            loggertesthelper.Logger(),
		HealthMonitor:     BadHealthMonitor{},
		StatusPort:        9878,
		Type:              "loggregator",
		StatusCredentials: []string{"user", "pass"},
	}

	go component.StartMonitoringEndpoints()

	req, err := http.NewRequest("GET", "http://localhost:9878/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, resp.StatusCode, 200)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, string(body), "bad")
}

func TestPanicWhenFailingToMonitorEndpoints(t *testing.T) {
	component := &Component{
		Logger:            loggertesthelper.Logger(),
		HealthMonitor:     GoodHealthMonitor{},
		StatusPort:        7879,
		Type:              "loggregator",
		StatusCredentials: []string{"user", "pass"},
	}

	finishChan := make(chan bool)

	go func() {
		err := component.StartMonitoringEndpoints()
		assert.NoError(t, err)
	}()
	time.Sleep(50 * time.Millisecond)

	go func() {
		//Monitoring a second time should fail because the port is already in use.
		err := component.StartMonitoringEndpoints()
		assert.Error(t, err)
		finishChan <- true
	}()

	<-finishChan
}

type testInstrumentable struct {
	name    string
	metrics []instrumentation.Metric
}

func (t testInstrumentable) Emit() instrumentation.Context {
	return instrumentation.Context{Name: t.name, Metrics: t.metrics}
}

func TestVarzRequiresBasicAuth(t *testing.T) {
	tags := map[string]interface{}{"tagName1": "tagValue1", "tagName2": "tagValue2"}
	component := &Component{
		Logger:            loggertesthelper.Logger(),
		HealthMonitor:     GoodHealthMonitor{},
		StatusPort:        1234,
		IpAddress:         "127.0.0.1",
		Type:              "loggregator",
		StatusCredentials: []string{"user", "pass"},
		Instrumentables: []instrumentation.Instrumentable{
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
	}

	go component.StartMonitoringEndpoints()

	req, err := http.NewRequest("GET", "http://localhost:1234/varz", nil)
	assert.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 401)
}

func TestVarzEndpoint(t *testing.T) {
	tags := map[string]interface{}{"tagName1": "tagValue1", "tagName2": "tagValue2"}
	component := &Component{
		Logger:            loggertesthelper.Logger(),
		HealthMonitor:     GoodHealthMonitor{},
		StatusPort:        1234,
		IpAddress:         "127.0.0.1",
		Type:              "loggregator",
		StatusCredentials: []string{"user", "pass"},
		Instrumentables: []instrumentation.Instrumentable{
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
	}

	go component.StartMonitoringEndpoints()

	req, err := http.NewRequest("GET", "http://localhost:1234/varz", nil)
	req.SetBasicAuth(component.StatusCredentials[0], component.StatusCredentials[1])
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

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
						"value": 2004,
					},
					map[string]interface{}{
						"name":  "queueLength",
						"value": 5,
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
						"value": 3,
					},
				},
			},
		},
	}

	var actualMap map[string]interface{}
	json.Unmarshal(body, &actualMap)
	assert.NotNil(t, actualMap["tags"])
	assert.Equal(t, expected["contexts"], actualMap["contexts"])
	assert.Equal(t, expected["name"], actualMap["name"])
	assert.Equal(t, expected["numCPUS"], actualMap["numCPUS"])
	assert.Equal(t, expected["numGoRoutines"], actualMap["numGoRoutines"])
	assert.NotNil(t, actualMap["memoryStats"])
	assert.NotEmpty(t, actualMap["memoryStats"])
}
