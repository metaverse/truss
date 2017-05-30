package middlewares

import (
	"golang.org/x/net/context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
)

// LabeledMiddleware will get passed the endpoint name when passed to
// WrapAllLabeledExcept, this can be used to write a generic metrics
// middleware which can send the endpoint name to the metrics collector.
type LabeledMiddleware func(string, endpoint.Endpoint) endpoint.Endpoint

// ErrorCounter is a LabeledMiddleware, when applied with WrapAllLabeledExcept name will be populated with the endpoint name, and such this middleware will
// report errors to the metric provider with the endpoint name. Feel free to
// copy this example middleware to your service.
func ErrorCounter(errCount metrics.Counter) LabeledMiddleware {
	return func(name string, in endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := in(ctx, req)
			if err != nil {
				errCount.With("endpoint", name).Add(1)
			}
			return resp, err
		}
	}
}

// Latency is a LabeledMiddleware, reporting the request time of and
// endpoint along with its name
func Latency(h metrics.Histogram) LabeledMiddleware {
	return func(name string, in endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			defer func(begin time.Time) {
				h.With("endpoint", name).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return in(ctx, req)
		}
	}
}
