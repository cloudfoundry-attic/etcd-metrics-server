package cfcomponent

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
	"strconv"
)

type Config struct {
	Syslog     string
	VarzPort   uint32
	VarzUser   string
	VarzPass   string
	NatsHost   string
	NatsPort   int
	NatsUser   string
	NatsPass   string
	MbusClient yagnats.NATSClient
}

var DefaultYagnatsClientProvider = func(logger *gosteno.Logger) yagnats.NATSClient {
	client := yagnats.NewClient()
	client.SetLogger(logger)
	return client
}

func (c *Config) Validate(logger *gosteno.Logger) (err error) {
	c.MbusClient = DefaultYagnatsClientProvider(logger)

	addr := c.NatsHost + ":" + strconv.Itoa(c.NatsPort)
	info := &yagnats.ConnectionInfo{
		Addr:     addr,
		Username: c.NatsUser,
		Password: c.NatsPass,
	}
	err = c.MbusClient.Connect(info)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not connect to NATS: %v", err.Error()))
	}
	return
}
