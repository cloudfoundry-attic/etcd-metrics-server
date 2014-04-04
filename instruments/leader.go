package instruments

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/metricz/instrumentation"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/gunk/urljoiner"
)

type Leader struct {
	statsEndpoint string

	logger *gosteno.Logger
}

var ErrRedirected = errors.New("redirected to leader")

func NewLeader(etcdAddr string, logger *gosteno.Logger) *Leader {
	return &Leader{
		statsEndpoint: urljoiner.Join(etcdAddr, "v2", "stats", "leader"),

		logger: logger,
	}
}

func (leader *Leader) Emit() instrumentation.Context {
	context := instrumentation.Context{
		Name:    "leader",
		Metrics: []instrumentation.Metric{},
	}

	var stats RaftFollowersStats

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return ErrRedirected
		},
	}

	resp, err := client.Get(leader.statsEndpoint)
	if err != nil {
		leader.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"leader.stat-collecting.failed",
		)

		return context
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&stats)
	if err != nil {
		leader.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"leader.stats.malformed",
		)

		return context
	}

	context.Metrics = []instrumentation.Metric{
		{
			Name:  "Followers",
			Value: len(stats.Followers),
		},
	}

	for name, follower := range stats.Followers {
		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "Latency",
			Value: follower.Latency.Current,
			Tags: map[string]interface{}{
				"follower": name,
			},
		})
	}

	return context
}
