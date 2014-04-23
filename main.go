package main

import (
	"flag"
	"log"
	"net/url"
	"strings"

	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"

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
	var err error

	flag.Parse()

	natsClient := yagnats.NewClient()

	natsMembers := []yagnats.ConnectionProvider{}

	for _, addr := range strings.Split(*natsAddresses, ",") {
		natsMembers = append(
			natsMembers,
			&yagnats.ConnectionInfo{addr, *natsUsername, *natsPassword},
		)
	}

	err = natsClient.Connect(&yagnats.ConnectionCluster{
		Members: natsMembers,
	})

	if err != nil {
		log.Fatalf("Error connecting to NATS: %s\n", err)
	}

	registrar := collector_registrar.New(natsClient)

	etcdURL := &url.URL{
		Scheme: *etcdScheme,
		Host:   *etcdAddress,
	}

	config := metrics_server.Config{
		JobName:  *jobName,
		EtcdURL:  etcdURL,
		Port:     *port,
		Username: *username,
		Password: *password,
		Index:    *index,
	}

	server := metrics_server.New(registrar, steno.NewLogger("etcd-metrics-server"), config)

	server.Start()

	select {}
}
