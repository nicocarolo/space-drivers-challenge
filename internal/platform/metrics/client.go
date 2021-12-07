package metrics

import "time"

type Collector interface {
	Inc(name string, tags []string)
	Count(name string, value int64, tags []string)
	Timing(name string, value time.Duration, tags []string)
	Gauge(name string, value float64, tags []string)
	Histogram(name string, value float64, tags []string)
}

type client struct{}

func (c client) Gauge(name string, value float64, tags []string) {
	// implement here calls to metric provider client
}
func (c client) Count(name string, value int64, tags []string) {
	// implement here calls to metric provider client
}
func (c client) Inc(name string, tags []string) {
	// implement here calls to metric provider client
}
func (c client) Histogram(name string, value float64, tags []string) {
	// implement here calls to metric provider client
}
func (c client) Timing(name string, value time.Duration, tags []string) {
	// implement here calls to metric provider client
}
