package instruments

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/gunk/urljoiner"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
)

type Server struct {
	statsEndpoint string

	logger *gosteno.Logger
}

func NewServer(etcdAddr string, logger *gosteno.Logger) *Server {
	return &Server{
		statsEndpoint: urljoiner.Join(etcdAddr, "v2", "stats", "self"),

		logger: logger,
	}
}

func (server *Server) Emit() instrumentation.Context {
	context := instrumentation.Context{
		Name: "server",
	}

	var stats RaftServerStats

	resp, err := http.Get(server.statsEndpoint)
	if err != nil {
		server.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"server.stat-collecting.failed",
		)

		return context
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&stats)
	if err != nil {
		server.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"server.stats.malformed",
		)

		return context
	}

	isLeader := 0
	if stats.State == "leader" {
		isLeader = 1
	}

	context.Metrics = []instrumentation.Metric{
		{
			Name:  "IsLeader",
			Value: isLeader,
		},
		{
			Name:  "SendingBandwidthRate",
			Value: stats.SendingBandwidthRate,
		},
		{
			Name:  "ReceivingBandwidthRate",
			Value: stats.RecvingBandwidthRate,
		},
		{
			Name:  "SendingRequestRate",
			Value: stats.SendingPkgRate,
		},
		{
			Name:  "ReceivingRequestRate",
			Value: stats.RecvingPkgRate,
		},
		{
			Name:  "SentAppendRequests",
			Value: stats.SendAppendRequestCnt,
		},
		{
			Name:  "ReceivedAppendRequests",
			Value: stats.RecvAppendRequestCnt,
		},
	}

	return context
}
