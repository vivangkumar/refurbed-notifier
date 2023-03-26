package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/vivangkumar/notify/cmd/internal/cli"
)

func run(ctx context.Context) error {
	cmd := cli.New()
	err := cmd.RunContext(ctx, os.Args)
	if err != nil {
		return fmt.Errorf("cli run: %w", err)
	}

	return nil
}

func main() {
	ctx := signalCtx(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	err := run(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func signalCtx(ctx context.Context, sig ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		c := make(chan os.Signal, len(sig))

		signal.Notify(c, sig...)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()

	return ctx
}
