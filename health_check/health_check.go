package health_check

import (
	"net"
	"time"

	"code.cloudfoundry.org/lager"
)

type HealthCheck struct {
	network string
	addr    string

	logger lager.Logger
}

func New(network, addr string, logger lager.Logger) *HealthCheck {
	return &HealthCheck{
		network: network,
		addr:    addr,

		logger: logger,
	}
}

func (check *HealthCheck) Ok() bool {
	conn, err := net.DialTimeout(check.network, check.addr, time.Second)
	if err != nil {
		check.logger.Error("health-check-failed", err)
		return false
	}

	conn.Close()

	return true
}
