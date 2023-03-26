package notification

import (
	"bytes"
	"fmt"
	"github.com/vivangkumar/notify/pkg/notification/internal/ratelimiter"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Client provides an interface to send notifications
// via HTTP calls to an upstream service.
type Client struct {
	// httpClient is the configured httpClient.
	//
	// Any client that satisfies the httpClient interface
	// can be passed.
	httpClient httpClient

	// rl is the configured rate limiter.
	//
	// If not configured, a default rate limiter is used.
	rl rateLimiter

	// cfg stores the internal config state.
	cfg config

	// msgs is the main message queue.
	msgs chan Message

	// errs is the channel via which clients can
	// read errors from.
	errs chan error

	// done is an internal channel to stop the
	// dispatcher go routine.
	done chan struct{}

	// wg keeps track of worker go routines.
	wg sync.WaitGroup

	// logger and metrics are for observability and monitoring.
	logger  *logrus.Logger
	metrics metrics
}

// NewClient constructs a Client with the given url and options
// along with a rate limit at which the client sends requests.
//
// The max buffer size can be specified to ensure the number of
// messages that the client will buffer before refusing to accept
// any more messages.
//
// Messages in the buffer will also be dealt with in accordance
// with the rate limit.
//
// If no options are specified, a Client is created with
// default configuration.
//
// Client.Start should be called to start consuming messages
// from the queue.
//
// Callers should also call Client.Stop to clean up resources
// from the Client.
func NewClient(url string, opts ...Opt) *Client {
	cfg := config{
		url:                   url,
		maxBufferSize:         defaultBufferSize,
		shutDownGraceDuration: defaultShutdownGraceDuration,
		maxConcurrency:        defaultConcurrency,
	}

	c := &Client{
		httpClient: &http.Client{},
		rl:         ratelimiter.New(defaultRateLimit, 1),
		cfg:        cfg,
		done:       make(chan struct{}),
		errs:       make(chan error),
		msgs:       make(chan Message, defaultBufferSize),
		wg:         sync.WaitGroup{},
		metrics:    newMetrics(false, prometheus.NewRegistry()),
		logger:     newLogger(false, defaultLogLevel),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Notify represents the main entry point to the Client.
//
// It accepts a variable number of messages and enqueues
// them to be fired off as HTTP requests.
//
// This method will likely never block depending on the buffer size.
//
// However, it will return an error if the enqueuing of the
// message fails and indicate to the caller that they must retry.
func (c *Client) Notify(msgs ...Message) error {
	for _, msg := range msgs {
		select {
		case c.msgs <- msg:
			c.logger.WithField("msg", msg).Debug("queuing message")
		default:
			c.logger.Info("failed to enqueue message")
			c.metrics.incrEnqueueFailures()

			return newEnqueueError(
				fmt.Errorf(
					"failed to enqueue message: %s", msg),
			)
		}
	}

	return nil
}

// Start begins the worker pool.
func (c *Client) Start() {
	c.logger.Info("starting message consuming")
	c.metrics.setClientMaxBufferSize(c.cfg.maxBufferSize)

	// Start the rate limiter
	c.rl.Start()

	for i := 0; i < c.cfg.maxConcurrency; i++ {
		go c.worker(i)
	}
}

func (c *Client) worker(i int) {
	c.wg.Add(1)
	defer c.wg.Done()

	c.logger.
		WithField("worker_num", i).
		Debug("starting worker")

	for {
		select {
		case msg, ok := <-c.msgs:
			if !ok {
				return
			}

			if err := c.retryRateLimit(); err != nil {
				c.sendError(
					fmt.Errorf("dropping msg: '%s': %w", msg, err),
				)
				continue
			}

			if err := c.send(msg); err != nil {
				c.logger.
					WithError(err).
					Errorln("failed to send notification")
				c.sendError(err)
			}
		case <-c.done:
			c.logger.Debug("stopping worker")
			return
		}
	}
}

// sendError attempts to send errors via the error channel
// in a non-blocking way.
//
// If there are no readers, errors are dropped.
func (c *Client) sendError(err error) {
	select {
	case c.errs <- err:
	default:
		c.logger.
			WithField("error", err.Error()).
			Debug("dropping error")
	}
}

// retryRateLimit attempts to send queue requests
// if the rate limiter is limiting us.
//
// It will give up if the client is stopped or
// if the retry duration has elapsed.
func (c *Client) retryRateLimit() error {
	allowed := c.rl.Add

	for {
		select {
		case <-c.done:
			return nil
		case <-time.After(defaultRateLimitRetryDuration):
			return fmt.Errorf("rate limit reached")
		default:
			if allowed() {
				return nil
			}
		}
	}
}

// Errors returns errors encountered from HTTP request.
//
// Callers can react to errors by reading from
// this channel.
//
// Errors are dropped if callers are not reading
// from this channel.
func (c *Client) Errors() <-chan error {
	return c.errs
}

// Stop gracefully shuts down the Client.
//
// It may return an error if the client cannot
// gracefully exit within the grace period.
func (c *Client) Stop() error {
	var err error

	c.rl.Stop()
	close(c.done)
	err = c.waitWithTimeout()
	close(c.errs)
	close(c.msgs)

	c.logger.WithError(err).Info("client stopped")

	return err
}

// MetricsRegistry returns the Prometheus metrics registry.
//
// This can be used by callers to report metrics via
// a Prometheus metrics endpoint or collect
// them in another way.
func (c *Client) MetricsRegistry() *prometheus.Registry {
	return c.metrics.registry()
}

// newLogger creates and configures a logger.
//
// By default, it uses the info log level.
// It always logs in a structured format.
func newLogger(enabled bool, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{})

	// Discard log output if disabled.
	if !enabled {
		logger.SetOutput(ioutil.Discard)
	}

	return logger
}

// waitWithTimeout ensures that the Client exits
// within the grace period it is configured with.
//
// This allows for a timely termination, as opposed to
// allowing the wait group to take an undetermined
// amount of time to complete.
func (c *Client) waitWithTimeout() error {
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(c.cfg.shutDownGraceDuration):
		return fmt.Errorf("shut down grace period exceeded")

	}
}

// send makes an HTTP POST request to the upstream service serving
// the url provided in the configuration of the Client.
//
// The body of the request is the message passed to it.
//
// Given we don't know what content type the upstream service
// will return response bodies in, this function does not
// read the response body.
//
// It detects errors by checking the status codes returned.
func (c *Client) send(msg Message) error {
	req, err := http.NewRequest(
		http.MethodPost,
		c.cfg.url,
		bytes.NewBuffer([]byte(msg)),
	)
	if err != nil {
		return fmt.Errorf("construct request: %w", err)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	c.metrics.measureHTTPLatency(start, resp.Status)

	return classifyStatus(resp.StatusCode, msg)
}

// classifyStatus inspects the request status to determine
// if it was successful or not.
//
// If it wasn't successful, it returns an appropriate error
// to indicate if clients can retry the request.
//
// The message that failed is also included as part of the
// error.
//
// All 5xx errors are retryable, while other errors are
// by default non retryable.
func classifyStatus(status int, msg Message) error {
	retryable := false

	switch s := status; {
	case is2XX(s):
		return nil
	case is5XX(status):
		retryable = true
	default:
	}

	return newRequestError(status, msg, retryable)
}

func is2XX(status int) bool {
	return status >= 200 && status < 300
}

func is5XX(status int) bool {
	return status >= 500
}
