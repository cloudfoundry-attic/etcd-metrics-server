package main

import (
	"flag"
	"net/url"
	"os"
	"strings"

	"github.com/cloudfoundry/yagnats"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"

	"github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/etcd-metrics-server/metrics_server"
	"github.com/cloudfoundry-incubator/metricz/collector_registrar"
)

var jobName = flag.String(
	"jobName",
	"etcd",
	"component name for collector",
)

var etcdScheme = flag.String(
	"etcdScheme",
	"http",
	"scheme to use for etcd requests",
)

var etcdAddress = flag.String(
	"etcdAddress",
	"127.0.0.1:4001",
	"etcd host:port to instrument",
)

var index = flag.Uint(
	"index",
	0,
	"index of the etcd job",
)

var port = flag.Int(
	"port",
	5678,
	"port to listen on",
)

var username = flag.String(
	"username",
	"",
	"basic auth username",
)

var password = flag.String(
	"password",
	"",
	"basic auth password",
)

var natsAddresses = flag.String(
	"natsAddresses",
	"127.0.0.1:4222",
	"comma-separated list of NATS addresses (ip:port)",
)

var natsUsername = flag.String(
	"natsUsername",
	"nats",
	"Username to connect to nats",
)

var natsPassword = flag.String(
	"natsPassword",
	"nats",
	"Password for nats user",
)

func main() {
	flag.Parse()

	logger := cf_lager.New("etcd-metrics-server")
	natsClient := initializeNatsClient(logger)

	cf_debug_server.Run()

	server := ifrit.Envoke(initializeServer(logger, natsClient))

	monitorProcess := ifrit.Envoke(sigmon.New(server))

	err := <-monitorProcess.Wait()
	if err != nil {
		os.Exit(1)
	}
}

func initializeServer(logger lager.Logger, natsClient yagnats.NATSClient) *metrics_server.MetricsServer {
	registrar := collector_registrar.New(natsClient)
	return metrics_server.New(registrar, logger, metrics_server.Config{
		JobName: *jobName,
		EtcdURL: &url.URL{
			Scheme: *etcdScheme,
			Host:   *etcdAddress,
		},
		Port:     *port,
		Username: *username,
		Password: *password,
		Index:    *index,
	})
}

func initializeNatsClient(logger lager.Logger) yagnats.NATSClient {
	natsClient := yagnats.NewClient()

	natsMembers := []yagnats.ConnectionProvider{}
	for _, addr := range strings.Split(*natsAddresses, ",") {
		natsMembers = append(
			natsMembers,
			&yagnats.ConnectionInfo{
				Addr:     addr,
				Username: *natsUsername,
				Password: *natsPassword,
			},
		)
	}

	err := natsClient.Connect(&yagnats.ConnectionCluster{
		Members: natsMembers,
	})
	if err != nil {
		logger.Fatal("failed-to-connect-to-nats", err)
	}

	return natsClient
}
