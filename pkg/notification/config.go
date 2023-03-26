package notification

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultShutdownGraceDuration  = 3 * time.Second
	defaultEnqueueRetryDuration   = 3 * time.Second
	defaultBufferSize             = 1000
	defaultLogLevel               = logrus.InfoLevel
	defaultRateLimit              = 100
	defaultRateLimitRetryDuration = 3 * time.Second
	defaultConcurrency            = 100
)

// config represents the configuration of the Notifier.
type config struct {
	// URL that requests will be sent to via
	// the HTTP client.
	url string

	// maxBufferSize specifies the size of the buffer
	// that holds messages should the client experience
	// an increase in rate of requests.
	maxBufferSize int

	// The additional time given to allow
	// the Notifier to exit gracefully.
	//
	// If this is exceeded, the Notifier is
	// shut down forcefully.
	shutDownGraceDuration time.Duration

	// maxConcurrency specifies the number of workers
	// to pick up new messages.
	maxConcurrency int
}
