package notification_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vivangkumar/notify/pkg/notification"
	"github.com/vivangkumar/notify/pkg/notification/internal/mocks"
)

func TestClient_Notify(t *testing.T) {
	t.Parallel()

	server := testServer(t)
	defer server.Close()

	mc := mocks.NewHTTPClient()
	client := notification.NewClient(
		server.URL+"/notification",
		notification.WithHTTPClient(mc),
	)
	client.Start()

	t.Run("single message", func(t *testing.T) {
		msg := "hello"

		err := client.Notify(msg)
		assert.Nil(t, err)

		assertChNoErrors(t, client.Errors(), 3*time.Second)
		assert.Equal(t, mc.CallCount(), 1)

		mc.Reset()
	})

	t.Run("multiple messages", func(t *testing.T) {
		msgs := []string{"msg1", "msg2"}

		err := client.Notify(msgs...)
		assert.Nil(t, err)

		assertChNoErrors(t, client.Errors(), 3*time.Second)
		assert.Equal(t, 2, mc.CallCount())
	})

	assert.Nil(t, client.Stop())
}

func TestClient_Notify_Errors(t *testing.T) {
	t.Parallel()

	server := testServer(t)
	defer server.Close()

	tests := []struct {
		url           string
		wantRetryable bool
	}{
		{
			url:           server.URL + "/error-400",
			wantRetryable: false,
		},
		{
			url:           server.URL + "/error-500",
			wantRetryable: true,
		},
	}

	for _, tc := range tests {
		client := notification.NewClient(tc.url)
		client.Start()

		err := client.Notify("hello")
		assert.Nil(t, err)

		err = <-client.Errors()
		assert.Error(t, err)

		var re requestError
		assert.True(t, errors.As(err, &re))
		assert.Equal(t, re.IsRetryable(), tc.wantRetryable)

		assert.Nil(t, client.Stop())
	}
}

func TestClient_Notify_Enqueue_Fail(t *testing.T) {
	t.Parallel()

	server := testServer(t)
	defer server.Close()

	client := notification.NewClient(
		server.URL+"/notification",
		notification.WithMaxBufferSize(1),
		notification.WithMaxConcurrency(1),
	)
	client.Start()

	// Buffer size is 1.
	assert.Nil(t, client.Notify("hello"))

	// This should fail
	err := client.Notify("hello2")
	assert.Error(t, err)

	var te temporaryError
	assert.True(t, errors.As(err, &te) && te.IsTemporary())

	assert.Nil(t, client.Stop())
}

func TestClient_Stop_Timeout(t *testing.T) {
	t.Parallel()

	server := testServer(t)
	defer server.Close()

	// Set a small grace period.
	client := notification.NewClient(
		server.URL+"/long",
		notification.WithShutDownGraceDuration(1*time.Second),
	)
	client.Start()

	// These will take a while so our requests will take a while
	err := client.Notify("msg1", "msg2")
	assert.Nil(t, err)

	<-time.After(1 * time.Second)

	err = client.Stop()
	assert.Error(t, err)
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/notification", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/error-400", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		return
	})

	mux.HandleFunc("/error-500", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	})

	mux.HandleFunc("/long", func(w http.ResponseWriter, req *http.Request) {
		<-time.After(3 * time.Second)
		w.WriteHeader(http.StatusCreated)
		return
	})

	return httptest.NewServer(mux)
}

func assertChNoErrors(t *testing.T, ch <-chan error, d time.Duration) {
	select {
	case err := <-ch:
		assert.Failf(t, "expected no error, but got ", err.Error())
	case <-time.After(d):
	}
}

type temporaryError interface {
	IsTemporary() bool
	RetryAfter() time.Duration
}

type requestError interface {
	IsRetryable() bool
}
