package metrics_server

import (
	"net/url"
	"os"

	"github.com/cloudfoundry-incubator/etcd-metrics-server/health_check"
	"github.com/cloudfoundry-incubator/metricz"
	"github.com/cloudfoundry-incubator/metricz/instrumentation"

	"github.com/cloudfoundry/gosteno"

	"github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
	"github.com/cloudfoundry-incubator/metricz/collector_registrar"
)

type MetricsServer struct {
	registrar collector_registrar.CollectorRegistrar
	logger    *gosteno.Logger
	config    Config
}

type Config struct {
	JobName string

	Index uint

	EtcdURL *url.URL

	Port     int
	Username string
	Password string
}

func New(registrar collector_registrar.CollectorRegistrar, logger *gosteno.Logger, config Config) *MetricsServer {
	return &MetricsServer{
		registrar: registrar,
		logger:    logger,
		config:    config,
	}
}

func (server *MetricsServer) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	component, err := metricz.NewComponent(
		server.logger,
		server.config.JobName,
		server.config.Index,
		health_check.New("tcp", server.config.EtcdURL.Host, server.logger),
		uint32(server.config.Port),
		[]string{server.config.Username, server.config.Password},
		[]instrumentation.Instrumentable{
			instruments.NewLeader(server.config.EtcdURL.String(), server.logger),
			instruments.NewServer(server.config.EtcdURL.String(), server.logger),
			instruments.NewStore(server.config.EtcdURL.String(), server.logger),
		},
	)

	if err != nil {
		return err
	}

	err = server.registrar.RegisterWithCollector(component)
	if err != nil {
		return err
	}

	close(ready)

	go func() {
		<-signals
		component.StopMonitoringEndpoints()
	}()

	return component.StartMonitoringEndpoints()
}

func (server *MetricsServer) start() error {
	component, err := metricz.NewComponent(
		server.logger,
		server.config.JobName,
		server.config.Index,
		health_check.New("tcp", server.config.EtcdURL.Host, server.logger),
		uint32(server.config.Port),
		[]string{server.config.Username, server.config.Password},
		[]instrumentation.Instrumentable{
			instruments.NewLeader(server.config.EtcdURL.String(), server.logger),
			instruments.NewServer(server.config.EtcdURL.String(), server.logger),
			instruments.NewStore(server.config.EtcdURL.String(), server.logger),
		},
	)

	if err != nil {
		return err
	}

	go component.StartMonitoringEndpoints()

	err = server.registrar.RegisterWithCollector(component)
	if err != nil {
		return err
	}

	return nil
}
