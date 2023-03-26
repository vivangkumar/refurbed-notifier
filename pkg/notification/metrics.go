package notification

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type metrics interface {
	setClientMaxBufferSize(size int)
	incrEnqueueFailures()
	measureHTTPLatency(start time.Time, status string)
	registry() *prometheus.Registry
}

// noop metrics is used when metrics are disabled.
type noopMetrics struct{}

func (n noopMetrics) setClientMaxBufferSize(_ int)             {}
func (n noopMetrics) incrEnqueueFailures()                     {}
func (n noopMetrics) measureHTTPLatency(_ time.Time, _ string) {}
func (n noopMetrics) registry() *prometheus.Registry           { return nil }

// clientMetrics contains a collection of prometheus metrics
// that can be reported if required.
type clientMetrics struct {
	// registry holds all the registered metrics
	// and their values.
	reg *prometheus.Registry

	// maxBufferSize reports the currently set
	// size of the client buffer.
	maxBufferSize prometheus.Gauge

	// enqueueFailures returns the number of
	// failures when attempting to queue messages.
	enqueueFailures prometheus.Counter

	// httpRequestLatency reports the request latency
	// of notification HTTP requests.
	//
	// Partitioned by status code.
	//
	// Histograms can be used to observe counts of
	// requests and errors (by status code).
	httpRequestLatency *prometheus.HistogramVec
}

func newMetrics(enabled bool, r *prometheus.Registry) metrics {
	if !enabled {
		return noopMetrics{}
	}

	m := &clientMetrics{
		reg: r,
		maxBufferSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "max_buffer_size",
			Help: "Reports the configured max buffer size.",
		}),
		enqueueFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notify_timeout_total",
			Help: "Reports the total number of failures when attempting to queue messages.",
		}),
		httpRequestLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_latency_duration_seconds",
			Help: "Reports the latency of notification HTTP requests.",
		}, []string{"status"}),
	}

	m.reg.MustRegister(
		m.maxBufferSize,
		m.enqueueFailures,
		m.httpRequestLatency,
	)

	return m
}

func (m *clientMetrics) setClientMaxBufferSize(cn int) {
	m.maxBufferSize.Set(float64(cn))
}

func (m *clientMetrics) incrEnqueueFailures() {
	m.enqueueFailures.Inc()
}

func (m *clientMetrics) measureHTTPLatency(start time.Time, status string) {
	m.httpRequestLatency.
		WithLabelValues(status).
		Observe(time.Since(start).Seconds())
}

func (m *clientMetrics) registry() *prometheus.Registry {
	return m.reg
}
