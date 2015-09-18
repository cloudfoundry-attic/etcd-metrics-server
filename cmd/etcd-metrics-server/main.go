package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/cloudfoundry-incubator/etcd-metrics-server/runners"
	"github.com/cloudfoundry-incubator/metricz/collector_registrar"
	"github.com/cloudfoundry/dropsonde"
	"github.com/cloudfoundry/gunk/diegonats"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
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

var metronAddress = flag.String(
	"metronAddress",
	"127.0.0.1:3457",
	"metron agent address",
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

var communicationTimeout = flag.Duration(
	"communicationTimeout",
	30*time.Second,
	"Timeout applied to all HTTP requests.",
)

var reportInterval = flag.Duration(
	"reportInterval",
	time.Minute,
	"interval on which to report metrics",
)

func main() {
	cf_debug_server.AddFlags(flag.CommandLine)
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	dropsonde.Initialize(*metronAddress, *jobName)
	cf_http.Initialize(*communicationTimeout)

	componentName := fmt.Sprintf("%s-metrics-server", *jobName)

	logger, reconfigurableSink := cf_lager.New(componentName)

	natsClient := diegonats.NewClient()
	natsClientRunner := diegonats.NewClientRunner(*natsAddresses, *natsUsername, *natsPassword, logger, natsClient)

	members := grouper.Members{
		{"nats-client", natsClientRunner},
		{"server", initializeServer(logger, natsClient)},
		{"metron-notifier", initializeMetronNotifier(logger)},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		members = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, reconfigurableSink)},
		}, members...)
	}

	group := grouper.NewOrdered(os.Interrupt, members)
	monitorProcess := ifrit.Invoke(sigmon.New(group))

	err := <-monitorProcess.Wait()
	if err != nil {
		os.Exit(1)
	}
}

func createEtcdURL() *url.URL {
	return &url.URL{
		Scheme: *etcdScheme,
		Host:   *etcdAddress,
	}
}

func initializeMetronNotifier(logger lager.Logger) *runners.PeriodicMetronNotifier {
	return runners.NewPeriodicMetronNotifier(createEtcdURL().String(), logger, *reportInterval)
}

func initializeServer(logger lager.Logger, natsClient diegonats.NATSClient) *runners.MetricsServer {
	registrar := collector_registrar.New(natsClient)
	return runners.New(registrar, logger, runners.Config{
		JobName:  *jobName,
		EtcdURL:  createEtcdURL(),
		Port:     *port,
		Username: *username,
		Password: *password,
		Index:    *index,
	})
}
