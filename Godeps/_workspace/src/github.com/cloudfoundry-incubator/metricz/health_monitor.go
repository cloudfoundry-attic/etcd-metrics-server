package metricz

type HealthMonitor interface {
	Ok() bool
}

type dummyHealthMonitor struct{}

func NewDummyHealthMonitor() HealthMonitor {
	return dummyHealthMonitor{}
}

func (hm dummyHealthMonitor) Ok() bool {
	return true
}
