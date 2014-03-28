package main

import (
	"flag"
	"log"
	"strings"

	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"

	"github.com/cloudfoundry-incubator/etcd-metrics/collector_registrar"
	"github.com/cloudfoundry-incubator/etcd-metrics/metrics_server"
)

var etcdMachine = flag.String(
	"etcdMachine",
	"http://127.0.0.1:4001",
	"etcd machine to instrument",
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

	config := metrics_server.Config{
		EtcdMachine: *etcdMachine,
		Port:        *port,
		Username:    *username,
		Password:    *password,
	}

	server := metrics_server.New(registrar, steno.NewLogger("etcd-metrics"), config)

	server.Start()

	select {}
}
