.PHONY: fmt lint build test

goos=linux
goarch=amd64

lint:
	go vet ./...
	golint ./...
fmt:
	go fmt -s -w .

build:
	go build -o build/notifier ./cmd

build-static:
	CGO_ENABLED=0 GOARCH=$(goarch) GOOS=$(goos) go build -o build/static-notifier -a -ldflags '-extldflags "-static"' ./cmd

test:
	go test -v ./...

