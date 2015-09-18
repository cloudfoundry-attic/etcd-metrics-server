package runners

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
	"github.com/cloudfoundry-incubator/metricz/instrumentation"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/pivotal-golang/lager"
)

type PeriodicMetronNotifier struct {
	etcdURL  string
	logger   lager.Logger
	interval time.Duration
}

func NewPeriodicMetronNotifier(
	etcdURL string,
	logger lager.Logger,
	interval time.Duration,
) *PeriodicMetronNotifier {

	return &PeriodicMetronNotifier{etcdURL, logger, interval}
}

func convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case uint64:
		return float64(v)
	case int:
		return float64(v)
	default:
		msg := fmt.Sprintf("invalid type %v", reflect.TypeOf(value).Name())
		panic(msg)
	}
}

func sendMetrics(instrument instrumentation.Instrumentable) {
	context := instrument.Emit()
	for _, metric := range context.Metrics {
		value := convertToFloat64(metric.Value)
		unit := GetMetricUnit(metric.Name)
		metrics.SendValue(metric.Name, value, unit)
	}
}

func (n *PeriodicMetronNotifier) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	instruments := []instrumentation.Instrumentable{
		instruments.NewLeader(n.etcdURL, n.logger),
		instruments.NewServer(n.etcdURL, n.logger),
		instruments.NewStore(n.etcdURL, n.logger),
	}

	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	close(ready)

	for {
		select {
		case <-ticker.C:

			for _, instrument := range instruments {
				sendMetrics(instrument)
			}

		case <-signals:
			return nil
		}
	}

	return nil
}
