package health_check

import (
	"net"
	"time"

	"github.com/cloudfoundry/gosteno"
)

type HealthCheck struct {
	network string
	addr    string

	logger *gosteno.Logger
}

func New(network, addr string, logger *gosteno.Logger) *HealthCheck {
	return &HealthCheck{
		network: network,
		addr:    addr,

		logger: logger,
	}
}

func (check *HealthCheck) Ok() bool {
	conn, err := net.DialTimeout(check.network, check.addr, time.Second)
	if err != nil {
		check.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"health-check.failed",
		)

		return false
	}

	conn.Close()

	return true
}
