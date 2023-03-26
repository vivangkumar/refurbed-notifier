package cli

import (
	"bufio"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"sync"

	"github.com/vivangkumar/notify/pkg/notification"
)

type notificationClient interface {
	Notify(msgs ...notification.Message) error
	Start()
	Stop() error
	Errors() <-chan error
	MetricsRegistry() *prometheus.Registry
}

type timedBuffer interface {
	Append(msgs ...notification.Message) error
	Close()
	FlushCh() <-chan []notification.Message
}

// notifier represents a component that makes use of
// the notification client.
//
// It reads stdin and forwards the lines read to a buffer
// which is flushed every "interval" that is configured.
type notifier struct {
	client notificationClient
	buffer timedBuffer

	// keep track of our go routines
	wg sync.WaitGroup

	logger *log.Logger
}

func newNotifier(
	client notificationClient,
	buffer timedBuffer,
	logger *log.Logger,
) *notifier {
	return &notifier{
		client: client,
		buffer: buffer,
		logger: logger,
	}
}

// start runs the notifier.
//
// It spins up two go routines:
//   One to scan for new lines from stdin.
//   One to log errors from the notification client.
//
// After that, it starts a blocking operation
// that waits on new messages to arrive so that they can
// be sent as notifications.
func (n *notifier) start(ctx *cli.Context) {
	go n.scan()
	go n.errors()
	n.client.Start()

	n.notify(ctx)
}

// scan reads from stdin.
//
// It will exit if all lines from stdin have been exhausted
// or if there is an error when reading from stdin.
//
// It continues to block in case of a long-running operation
// or waiting for user input, in which case it expects an EOF
// to gracefully exit.
func (n *notifier) scan() {
	n.logger.Println("reading stdin...")

	n.wg.Add(1)
	defer n.wg.Done()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		err := n.buffer.Append(scanner.Text())
		if err != nil {
			n.logger.Printf("buffer append: %s\n", err.Error())
		}
	}

	// Kill the program if there is an error reading
	// from stdin.
	if err := scanner.Err(); err != nil {
		log.Fatalf("scanner: %s", err.Error())
	}
}

// errors logs errors returned from the client.
func (n *notifier) errors() {
	n.wg.Add(1)
	defer n.wg.Done()

	for err := range n.client.Errors() {
		n.logger.Printf("client error: %s\n", err.Error())
	}
}

// notify waits for messages from the buffer
// (which are added to by the scanner) and sends them
// via the client.
//
// It is also responsible for receiving the context done event.
func (n *notifier) notify(ctx *cli.Context) {
	n.logger.Println("waiting for messages...")

	for {
		select {
		case msgs := <-n.buffer.FlushCh():
			n.logger.Printf("sending messages: %v\n", msgs)
			err := n.client.Notify(msgs...)
			if err != nil {
				n.logger.Printf("message queue error: %s\n", err.Error())
			}
		case <-ctx.Done():
			n.logger.Printf("received interrupt...")
			return
		}
	}
}

// stop gracefully shuts down the client
// and waits for go routines to finish.
//
// An error here is only indicative that
// we couldn't shut down the client gracefully.
func (n *notifier) stop() error {
	n.buffer.Close()
	n.logger.Println("stopping notifier...")

	err := n.client.Stop()
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	n.wg.Wait()
	return nil
}
