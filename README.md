# notify

This project implements an HTTP notification sender and a command line tool
that makes use of the sender.

Package `pkg` contains the HTTP notification sender library.

It is implemented using the worker pool pattern where messages are sent over
a channel that workers action by picking them up and sending HTTP requests to
the configured URL.

In addition to this, the client also implements a rate limiter to ensure
that even though there maybe several workers to process requests, we're not
exceeding the rate limit of the service downstream.

The library also buffers requests upto a max size to ensure that requests in
the queue are gracefully handled.

Package `cmd` contains the command line tool that reads from STDIN and
sends messages via the notification client in `pkg`.

Ideally, this would be a separate library/ package, but for ease of setup and review
as an interview task, I've included it as part of this.

## Build

To build, run `make build` (also lints and formats the code).

This will create a binary for the CLI in the build `build` folder.

## Test

```bash
make test
```

Note that some of these tests involve waiting for a while due to the nature
of concurrent tests.

## Run

To run the CLI,

`./notifier --url <url> < message.txt `

```bash
NAME:
   notifier - sends notifications from stdin

USAGE:
   notifier [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --url value, -u value                url to send notifications to
   --interval value, -i value           interval after which notifications are sent (default: 5s)
   --verbose, -v                        enables logging (default: false)
   --max-buffer-size value, --bs value  max buffer size between notification sends (default: 1000)
   --max-rps value, --rps value         max requests per second the client can send (default: 100)
   --max-concurrency value, --cn value  max concurrency of the notifier client (default: 100)
   --help, -h                           show help
```

The CLI can either:
1. Accept input redirected from files, in which case it will exit once all the file
   data has been ingested as notifications.
	 If the file is large the program can be interrupted with `ctrl-c` which will
   gracefully shut down the cli.
2. Accept input from the user, waiting for new messages to arrive.
   In this case, an EOF (`ctrl-d`) must be sent so that the program followed 
   by an interrupt so that we stop reading from stdin and gracefully exit.

## Decision Log & Thoughts

1. I implemented this as I would a public library that might be open source.
   This meant thinking about some safety features as well as it being well documented.
2. This has also been implemented with the mindset of it being deployed to production
   and as such is well tested and implements graceful shutdown.
   When deploying something to production, I always think about monitoring & observability.
   With this in mind, the library also exposes a logging interface that can be configured
   and returns a Prometheus registry for consumption by callers.
3. The library provides configuration options using the functional options pattern.
4. The library also makes use of the worker pool pattern to bound concurrency to 
   not make too many requests to the upstream service.
5. The library implements backpressure by letting clients know when the notifier
   is being overwhelmed with requests.
   The library presents behavioural errors that can be tested for attributes such as
   `IsTemporary` and `IsRetryable` making clients aware of when operations can be retried.
    In addition, the `RetryAfter` method specifies when clients can retry requests.
6. I tried to mostly make use of the stdlib, only using a few external packages
   for assertions, logging, metrics and to build the CLI. Mostly to save time, given
   there were no restrictions on libraries that can be used.
7. It took me roughly 5 hours to implement this. I managed to get a working system in a few hours
   but given it required production readiness, I took additional time to implement some
   production ready features such as logging and metrics.
8. My interpretation of the CLI was that it should "buffer" messages until the notification
   interval is reached, after which all the buffered messages are sent via the notifier
   in batches.
   This can likely be simplified and the notifier can be called more frequently.
9. Currently, the `Notify` method is non-blocking for the most part, but will return
   an error if the message cannot be queued to ensure that we do not indefinitely block.
   The client also buffers requests depending on configuration. Once this buffer is full,
   It will fail to queue requests.
   However, there is always the question of "how many" messages to hold.

## Improvements

1. The library returns errors with the intention of making callers aware of when
   the upstream service might be overwhelmed.
   These are implemented as error behaviours (`IsTemporary`, `IsRetryable` and `RetryAfter`).
   The cli doesn't make use of the `RetryAfter` method to retry requests (which can be implemented)
2. Alternatively, we can also allow the worker pool to grow and shrink with demand
   making it more dynamic.
3. Tests for the cli. I didn't have enough time to implement these.
4. The tests might need some slight improvement (given they're concurrent)
5. I wanted to build a Docker image for the program, but there are some issues with my
   docker M1 installation and I couldn't quite verify it.
6. Return message ID's for messages that couldn't be queued so that callers may handle retries
   by keying against the ID when storing them for later retry.