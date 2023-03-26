package timedbuffer

import (
	"fmt"
	"sync"
	"time"

	"github.com/vivangkumar/notify/pkg/notification"
)

// Buffer represents a "buffer" that holds a buffer
// of messages which are flushed in accordance with the
// flush interval.
//
// It will gather values that are to be published until a
// single tick is detected, after which these values are
// flushed by sending them over the flush channel.
//
// Close should be called to release resources to avoid
// leaking the ticker.
type Buffer struct {
	// ticker ticks every time the duration interval
	// passes.
	ticker *time.Ticker

	// flushCh is the channel over which a message batch
	// is flushed.
	flushCh chan []notification.Message

	// stopCh is used to stop the gatherer
	// when the buffer is closed.
	stopCh chan struct{}

	// buffer holds the buffered messages that
	// have to be sent to the notifier.
	//
	// Note that this can grow unbounded if
	// the interval is set large enough.
	buffer []notification.Message

	// max size that the buffer can grow to.
	//
	// Messages added to the buffer that exceed the size
	// will be dropped.
	size int
	m    sync.Mutex
}

// New constructs a new Buffer with the
// specified interval and size.
//
// It is important to specify an adequate interval
// to flush small batches of messages rather than
// let the buffer build up.
//
// It spawns a go routine that keeps track of the
// timer and gathers the messages added to the buffer.
func New(interval time.Duration, size int) *Buffer {
	t := time.NewTicker(interval)

	b := &Buffer{
		ticker:  t,
		flushCh: make(chan []notification.Message),
		stopCh:  make(chan struct{}),
		m:       sync.Mutex{},
		size:    size,
	}

	// Start gathering messages.
	go b.gather()

	return b
}

// gather waits on a new tick from ticker.
//
// On a new tick, the gathered messages are flushed to
// flushCh.
//
// If no new messages have been added to the buffer,
// the buffer is not flushed.
//
// Sends to the flushCh will never block.
// Message batches will be dropped if there is no receiver.
func (b *Buffer) gather() {
	defer close(b.flushCh)

	for {
		select {
		case <-b.ticker.C:
			b.flush()
		case <-b.stopCh:
			return
		}
	}
}

// flush flushes all messages to flushCh.
func (b *Buffer) flush() {
	// Copy the buffer and release
	// the lock.
	b.m.Lock()
	buf := b.buffer
	b.buffer = nil
	b.m.Unlock()

	if len(buf) == 0 {
		return
	}

	// Send the batch to the flushCh.
	//
	// This is non-blocking so if there is
	// no reader for the channel, batches
	// will be discarded.
	select {
	case b.flushCh <- buf:
	default:
	}
	buf = nil
}

// Append adds appends messages to the buffer.
func (b *Buffer) Append(msgs ...notification.Message) error {
	b.m.Lock()
	defer b.m.Unlock()

	if len(msgs) == 0 || msgs == nil {
		return nil
	}

	if len(b.buffer)+len(msgs) > b.size {
		return fmt.Errorf("max buffer size of %d exceeded", b.size)
	}

	b.buffer = append(b.buffer, msgs...)

	return nil
}

// FlushCh returns the channel to which message batches are
// flushed to
func (b *Buffer) FlushCh() <-chan []notification.Message {
	return b.flushCh
}

// Close stops the ticker and releases associated resources.
func (b *Buffer) Close() {
	b.ticker.Stop()
	close(b.stopCh)
}
