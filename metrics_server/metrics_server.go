package metrics_server

import (
	"github.com/cloudfoundry-incubator/etcd-metrics-server/health_check"
	"net/url"
	"strconv"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"

	"github.com/cloudfoundry-incubator/etcd-metrics-server/collector_registrar"
	"github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
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

func (server *MetricsServer) Start() error {
	url, err := url.Parse(server.config.EtcdMachine)
	if err != nil {
		return err
	}

	component, err := cfcomponent.NewComponent(
		server.logger,
		"etcd",
		0,
		health_check.New("tcp", url.Host, server.logger),
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
		"etcd-metrics-server.listening",
	)

	err = server.registrar.RegisterWithCollector(component)
	if err != nil {
		return err
	}

	return nil
}
