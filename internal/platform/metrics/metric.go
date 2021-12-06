package metrics

import (
	"context"
	"time"
)

var DefaultTracer = client{}

func Inc(ctx context.Context, name string, tags []string) {
	getClient(ctx).Inc(name, tags)
}

func Count(ctx context.Context, name string, value int64, tags []string) {
	getClient(ctx).Count(name, value, tags)
}

func Timing(ctx context.Context, name string, value time.Duration, tags []string) {
	getClient(ctx).Timing(name, value, tags)
}

func Gauge(ctx context.Context, name string, value float64, tags []string) {
	getClient(ctx).Gauge(name, value, tags)
}

func Histogram(ctx context.Context, name string, value float64, tags []string) {
	getClient(ctx).Histogram(name, value, tags)
}

type collectorCtxKey struct{}

func getClient(ctx context.Context) Collector {
	// it should exist a middleware where the collector is inyected into context, then application can trace without
	// using DefaultTracer
	l, ok := ctx.Value(collectorCtxKey{}).(Collector)
	if ok {
		return l
	}

	return DefaultTracer
}
