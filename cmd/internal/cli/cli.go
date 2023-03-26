package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/vivangkumar/notify/cmd/internal/timedbuffer"
	"github.com/vivangkumar/notify/pkg/notification"
)

const (
	urlFlag            = "url"
	intervalFlag       = "interval"
	verboseFlag        = "verbose"
	maxBufferSizeFlag  = "max-buffer-size"
	maxRpsFlag         = "max-rps"
	maxConcurrencyFlag = "max-concurrency"
)

// New creates a new command line interface that allows
// sending notifications from stdin.
func New() *cli.App {
	return &cli.App{
		Name:  "notifier",
		Usage: "sends notifications from stdin",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     urlFlag,
				Aliases:  []string{"u"},
				Required: true,
				Usage:    "url to send notifications to",
			},
			&cli.DurationFlag{
				Name:    intervalFlag,
				Aliases: []string{"i"},
				Value:   5 * time.Second,
				Usage:   "interval after which notifications are sent",
			},
			&cli.BoolFlag{
				Name:    verboseFlag,
				Aliases: []string{"v"},
				Value:   false,
				Usage:   "enables logging",
			},
			&cli.IntFlag{
				Name:    maxBufferSizeFlag,
				Aliases: []string{"bs"},
				Value:   1000,
				Usage:   "max buffer size between notification sends",
			},
			&cli.Uint64Flag{
				Name:    maxRpsFlag,
				Aliases: []string{"rps"},
				Value:   100,
				Usage:   "max requests per second the client can send",
			},
			&cli.IntFlag{
				Name:    maxConcurrencyFlag,
				Aliases: []string{"cn"},
				Value:   100,
				Usage:   "max concurrency of the notifier client",
			},
		},
		Action: run,
	}
}

// run is the main entry point to the program.
func run(ctx *cli.Context) error {
	url := ctx.String(urlFlag)
	interval := ctx.Duration(intervalFlag)
	verbose := ctx.Bool(verboseFlag)
	maxBufferSize := ctx.Int(maxBufferSizeFlag)
	maxRps := ctx.Uint64(maxRpsFlag)
	maxConcurrency := ctx.Int(maxConcurrencyFlag)

	logger := log.New()
	logger.SetFormatter(&log.TextFormatter{})
	logger.SetOutput(ioutil.Discard)

	clientOpts := []notification.Opt{
		notification.WithMaxBufferSize(maxBufferSize),
		notification.WithMaxRpsAndRefill(maxRps, 1),
		notification.WithMaxConcurrency(maxConcurrency),
	}
	if verbose {
		clientOpts = append(
			clientOpts,
			notification.WithLoggingEnabled(log.InfoLevel),
		)
		logger.SetOutput(os.Stderr)
	}

	client := notification.NewClient(url, clientOpts...)
	buffer := timedbuffer.New(interval, maxBufferSize)

	notifier := newNotifier(client, buffer, logger)
	notifier.start(ctx)

	err := notifier.stop()
	if err != nil {
		return fmt.Errorf("notifier: %w", err)
	}

	return nil
}
