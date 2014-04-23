package metricz

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/cloudfoundry-incubator/metricz/auth"
	"github.com/cloudfoundry-incubator/metricz/instrumentation"
	"github.com/cloudfoundry-incubator/metricz/localip"
	"github.com/cloudfoundry/gosteno"
)

type Component struct {
	name              string //Used by the collector to find data processing class
	ipAddress         string
	healthMonitor     HealthMonitor
	index             uint
	uuid              string
	statusPort        uint32
	statusCredentials []string
	instrumentables   []instrumentation.Instrumentable

	logger *gosteno.Logger

	listener  net.Listener
	quitChan  chan bool
	startChan chan bool
}

const (
	username = iota
	password
)

func NewComponent(
	logger *gosteno.Logger,
	name string,
	index uint,
	heathMonitor HealthMonitor,
	statusPort uint32,
	statusCreds []string,
	instrumentables []instrumentation.Instrumentable,
) (Component, error) {
	ip, err := localip.LocalIP()
	if err != nil {
		return Component{}, err
	}

	if statusPort == 0 {
		statusPort, err = localip.LocalPort()
		if err != nil {
			return Component{}, err
		}
	}

	if len(statusCreds) == 0 || statusCreds[username] == "" || statusCreds[password] == "" {
		randUser := make([]byte, 42)
		randPass := make([]byte, 42)
		rand.Read(randUser)
		rand.Read(randPass)
		en := base64.URLEncoding
		user := en.EncodeToString(randUser)
		pass := en.EncodeToString(randPass)

		statusCreds = []string{user, pass}
	}

	return Component{
		name:              name,
		ipAddress:         ip,
		index:             index,
		healthMonitor:     heathMonitor,
		statusPort:        statusPort,
		statusCredentials: statusCreds,
		instrumentables:   instrumentables,

		logger: logger,

		quitChan:  make(chan bool, 1),
		startChan: make(chan bool, 1),
	}, nil
}

func (c *Component) StartMonitoringEndpoints() error {
	mux := http.NewServeMux()
	auth := auth.NewBasicAuth("Realm", c.statusCredentials)
	mux.HandleFunc("/healthz", healthzHandlerFor(c))
	mux.HandleFunc("/varz", auth.Wrap(varzHandlerFor(c)))

	c.logger.Debugd(
		map[string]interface{}{
			"uuid":     c.uuid,
			"ip":       c.ipAddress,
			"port":     c.statusPort,
			"username": c.statusCredentials[username],
			"password": c.statusCredentials[password],
		},
		"component.varz.start",
	)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", c.ipAddress, c.statusPort))
	if err != nil {
		return err
	}

	c.listener = listener
	c.startChan <- true

	server := &http.Server{Handler: mux}
	err = server.Serve(listener)
	select {
	case <-c.quitChan:
		return nil
	default:
		return err
	}
}

func (c *Component) StopMonitoringEndpoints() {
	<-c.startChan
	c.quitChan <- true
	c.listener.Close()
}

func (c *Component) Name() string {
	return c.name
}

func (c *Component) Index() uint {
	return c.index
}

func (c *Component) UUID() string {
	return c.uuid
}

func (c Component) URL() *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(c.ipAddress, fmt.Sprintf("%d", c.statusPort)),
		User:   url.UserPassword(c.statusCredentials[0], c.statusCredentials[1]),
	}
}

func healthzHandlerFor(c *Component) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if c.healthMonitor.Ok() {
			fmt.Fprintf(w, "ok")
		} else {
			fmt.Fprintf(w, "bad")
		}
	}
}

func varzHandlerFor(c *Component) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		message, err := instrumentation.NewVarzMessage(c.name, c.instrumentables)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}

		json, err := json.Marshal(message)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		w.Write(json)
	}
}
