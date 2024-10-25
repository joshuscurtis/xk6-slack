SHELL := /bin/bash

.PHONY: test
test:
	go test -v -race ./...

.PHONY: build
build:
	xk6 build --with github.com/$(shell git config --get remote.origin.url | sed 's/.*:\(.*\)\.git/\1/')@latest

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: setup
setup:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: clean
clean:
	rm -f k6
	go clean -cache
