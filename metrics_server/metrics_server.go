package metrics_server

import (
	"strconv"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"

	"github.com/cloudfoundry-incubator/etcd-metrics/collector_registrar"
	"github.com/cloudfoundry-incubator/etcd-metrics/instruments"
)

type MetricsServer struct {
	registrar collector_registrar.CollectorRegistrar
	logger    *gosteno.Logger
	config    Config
}

type Config struct {
	EtcdMachine string
	Port        int
	Username    string
	Password    string
}

func New(registrar collector_registrar.CollectorRegistrar, logger *gosteno.Logger, config Config) *MetricsServer {
	return &MetricsServer{
		registrar: registrar,
		logger:    logger,
		config:    config,
	}
}

func (server *MetricsServer) Ok() bool {
	return true
}

func (server *MetricsServer) Start() error {
	component, err := cfcomponent.NewComponent(
		server.logger,
		"etcd",
		0,
		server,
		uint32(server.config.Port),
		[]string{server.config.Username, server.config.Password},
		[]instrumentation.Instrumentable{
			instruments.NewLeader(server.config.EtcdMachine, server.logger),
			instruments.NewServer(server.config.EtcdMachine, server.logger),
			instruments.NewStore(server.config.EtcdMachine, server.logger),
		},
	)

	if err != nil {
		return err
	}

	go component.StartMonitoringEndpoints()

	server.logger.Infod(
		map[string]interface{}{
			"ip":       component.IpAddress,
			"port":     strconv.Itoa(int(component.StatusPort)),
			"username": component.StatusCredentials[0],
			"password": component.StatusCredentials[1],
		},
		"etcd-metrics.server.listening",
	)

	err = server.registrar.RegisterWithCollector(component)
	if err != nil {
		return err
	}

	return nil
}
