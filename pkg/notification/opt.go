package notification

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/vivangkumar/notify/pkg/notification/internal/ratelimiter"
)

type httpClient interface {
	Do(r *http.Request) (*http.Response, error)
}

type rateLimiter interface {
	Start()
	Add() bool
	Stop()
}

// Opt represents options that can be passed to the Client.
// These can be used to configure the Client.
type Opt func(c *Client)

// WithHTTPClient allows setting the HTTP client
// for the notifier.
//
// This can be leverage to pass in custom transports,
// or to set a timeout for the http client.
func WithHTTPClient(cli httpClient) Opt {
	return func(c *Client) {
		c.httpClient = cli
	}
}

// WithMaxBufferSize sets the max number of messages
// that the client can buffer.
//
// This allows the client to deal with an increase in
// request rate.
func WithMaxBufferSize(size int) Opt {
	return func(c *Client) {
		c.cfg.maxBufferSize = size
		c.msgs = make(chan Message, size)
	}
}

// WithRateLimiter sets the rate limiter for the client.
func WithRateLimiter(rl rateLimiter) Opt {
	return func(c *Client) {
		c.rl = rl
	}
}

// WithMaxRpsAndRefill sets the rps for the rate limiter along
// with the tokens that are to be refilled.
func WithMaxRpsAndRefill(rps uint64, refill uint64) Opt {
	return func(c *Client) {
		c.rl = ratelimiter.New(rps, refill)
	}
}

// WithShutDownGraceDuration sets the grace period
// allowed for the Client to shut down.
func WithShutDownGraceDuration(d time.Duration) Opt {
	return func(c *Client) {
		c.cfg.shutDownGraceDuration = d
	}
}

// WithLoggingEnabled enables and sets the log level for the Client.
//
// It is turned off by default.
func WithLoggingEnabled(ll log.Level) Opt {
	return func(c *Client) {
		c.logger = newLogger(true, ll)
	}
}

// WithMetrics allows passing in a custom registry
// and allow metric collection.
// It is disabled by default.
//
// By default, a registry is created if not set.
func WithMetrics(r *prometheus.Registry) Opt {
	return func(c *Client) {
		c.metrics = newMetrics(true, r)
	}
}

// WithMaxConcurrency sets the max number of workers available
// to process new messages.
//
// It is set to 100 by default.
func WithMaxConcurrency(cn int) Opt {
	return func(c *Client) {
		c.cfg.maxConcurrency = cn
	}
}
