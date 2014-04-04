package metricz

type HealthMonitor interface {
	Ok() bool
}
