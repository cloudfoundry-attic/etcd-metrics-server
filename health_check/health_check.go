package health_check

import (
	"net"

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
	conn, err := net.Dial(check.network, check.addr)
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
