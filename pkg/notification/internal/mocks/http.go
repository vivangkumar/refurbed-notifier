// Package mocks is a minimal mock/ double package.
package mocks

import (
	"net/http"
	"sync"
)

// HTTPClient is a mock HTTP client implementation
// that works as a double.
//
// It captures the number of calls made to Do.
//
// It is safe for concurrent use.
type HTTPClient struct {
	callCount int
	m         sync.Mutex

	defaultClient *http.Client
}

// NewHTTPClient constructs a new mock HTTP client.
//
// Reset must be called to reset the internal state.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		defaultClient: http.DefaultClient,
	}
}

// Do makes an HTTP request by calling the underlying client.
//
// It also increments the call count.
func (mc *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	mc.m.Lock()
	mc.callCount++
	mc.m.Unlock()

	return mc.defaultClient.Do(req)
}

// CallCount returns the number of calls made to the client.
func (mc *HTTPClient) CallCount() int {
	mc.m.Lock()
	defer mc.m.Unlock()

	return mc.callCount
}

// Reset resets the mock.
func (mc *HTTPClient) Reset() {
	mc.m.Lock()
	defer mc.m.Unlock()

	mc.callCount = 0
}
