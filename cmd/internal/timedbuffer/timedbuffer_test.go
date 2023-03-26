package timedbuffer_test

import (
	"sync"
	"testing"
	"time"

	"github.com/vivangkumar/notify/cmd/internal/timedbuffer"
	"github.com/vivangkumar/notify/pkg/notification"

	"github.com/stretchr/testify/assert"
)

func TestTimedBuffer_SingleMessage(t *testing.T) {
	t.Parallel()

	tb := timedbuffer.New(3*time.Second, 100)

	err := tb.Append("hello")
	assert.Nil(t, err)

	batch := <-tb.FlushCh()
	assert.Equal(t, len(batch), 1)
	assert.Equal(t, batch[0], "hello")

	tb.Close()
}

func TestTimedBuffer_MultipleMessages(t *testing.T) {
	t.Parallel()

	tb := timedbuffer.New(3*time.Second, 100)

	msgs := []string{"a", "b", "c", "d", "e"}

	err := tb.Append(msgs...)
	assert.Nil(t, err)

	batch := <-tb.FlushCh()
	assert.Equal(t, len(batch), len(msgs))
	assert.ElementsMatch(t, batch, msgs)

	tb.Close()
}

func TestTimedBuffer_ContinuousBatches(t *testing.T) {
	t.Parallel()

	tb := timedbuffer.New(5*time.Second, 100)
	var wg sync.WaitGroup

	done := make(chan struct{}, 1)

	wg.Add(1)
	go func(t *testing.T, wg *sync.WaitGroup) {
		p := time.NewTicker(1 * time.Second)
		defer p.Stop()
		defer wg.Done()

		for {
			select {
			case <-p.C:
				err := tb.Append("msg")
				assert.Nil(t, err)
			case <-done:
				return
			}
		}
	}(t, &wg)

	wg.Add(1)
	go func(t *testing.T, wg *sync.WaitGroup) {
		defer wg.Done()

		var batches [][]notification.Message
		for msgs := range tb.FlushCh() {
			batches = append(batches, msgs)
		}

		assert.Equal(t, len(batches), 5)
	}(t, &wg)

	<-time.After(25 * time.Second)
	tb.Close()

	done <- struct{}{}

	wg.Wait()
}

func TestTimedBuffer_OverBufferSize(t *testing.T) {
	t.Parallel()

	tb := timedbuffer.New(3*time.Second, 5)

	msgs := []string{"1", "2", "3", "4", "5"}
	for _, msg := range msgs {
		err := tb.Append(msg)
		assert.Nil(t, err)
	}

	err := tb.Append("msg")
	assert.Error(t, err)

	flushed := <-tb.FlushCh()
	tb.Close()

	assert.Equal(t, len(flushed), 5)
	assert.ElementsMatch(t, msgs, flushed)
}

func TestTimedBuffer_NoMessages(t *testing.T) {
	t.Parallel()

	tb := timedbuffer.New(3*time.Second, 10)
	defer tb.Close()

	select {
	case <-tb.FlushCh():
		assert.Fail(t, "expected no messages")
	case <-time.After(5 * time.Second):
	}
}
