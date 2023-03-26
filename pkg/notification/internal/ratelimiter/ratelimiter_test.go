package ratelimiter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vivangkumar/notify/pkg/notification/internal/ratelimiter"
)

func TestRateLimiter_Add_TokensAvailable(t *testing.T) {
	r := ratelimiter.New(10, 1)
	r.Start()
	defer r.Stop()

	assert.True(t, r.Add())
}

func TestRateLimiter_Add_NoTokens(t *testing.T) {
	r := ratelimiter.New(1, 0)
	r.Start()
	defer r.Stop()

	assert.True(t, r.Add())
	assert.False(t, r.Add())

}

func TestRateLimiter_RefreshToken(t *testing.T) {
	r := ratelimiter.New(1, 1)
	r.Start()
	defer r.Stop()

	assert.True(t, r.Add())
	assert.False(t, r.Add())

	<-time.After(2 * time.Second)
	assert.True(t, r.Add())
}
