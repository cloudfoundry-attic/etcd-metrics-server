package instruments

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/gunk/urljoiner"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
)

type Store struct {
	statsEndpoint string
	keysEndpoint  string

	logger *gosteno.Logger
}

func NewStore(etcdAddr string, logger *gosteno.Logger) *Store {
	return &Store{
		statsEndpoint: urljoiner.Join(etcdAddr, "v2", "stats", "store"),
		keysEndpoint:  urljoiner.Join(etcdAddr, "v2", "keys", "/"),

		logger: logger,
	}
}

func (store *Store) Emit() instrumentation.Context {
	context := instrumentation.Context{
		Name: "store",
	}

	var stats map[string]uint64

	statsResp, err := http.Get(store.statsEndpoint)
	if err != nil {
		store.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"store.stat-collecting.failed",
		)

		return context
	}

	defer statsResp.Body.Close()

	err = json.NewDecoder(statsResp.Body).Decode(&stats)
	if err != nil {
		store.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"store.stats.malformed",
		)

		return context
	}

	keysResp, err := http.Get(store.keysEndpoint)
	if err != nil {
		store.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"store.read.failed",
		)

		return context
	}

	defer keysResp.Body.Close()

	etcdIndexHeader := keysResp.Header.Get("X-Etcd-Index")
	raftIndexHeader := keysResp.Header.Get("X-Raft-Index")
	raftTermHeader := keysResp.Header.Get("X-Raft-Term")

	etcdIndex, err := strconv.ParseUint(etcdIndexHeader, 10, 0)
	if err != nil {
		store.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"store.etcd-index.malformed",
		)

		return context
	}

	raftIndex, err := strconv.ParseUint(raftIndexHeader, 10, 0)
	if err != nil {
		store.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"store.raft-index.malformed",
		)

		return context
	}

	raftTerm, err := strconv.ParseUint(raftTermHeader, 10, 0)
	if err != nil {
		store.logger.Errord(
			map[string]interface{}{
				"error": err.Error(),
			},
			"store.raft-term.malformed",
		)

		return context
	}

	context.Metrics = []instrumentation.Metric{
		{
			Name:  "EtcdIndex",
			Value: etcdIndex,
		},
		{
			Name:  "RaftIndex",
			Value: raftIndex,
		},
		{
			Name:  "RaftTerm",
			Value: raftTerm,
		},
	}

	for name, val := range stats {
		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  strings.ToUpper(name[0:1]) + name[1:],
			Value: val,
		})
	}

	return context
}
