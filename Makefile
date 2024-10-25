SHELL := /bin/bash

.PHONY: test
test:
	go test -v -race ./...

.PHONY: build
build:
	xk6 build --with github.com/joshuscurtis/xk6-slack@latest

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: setup
setup:
	go mod download
	go install github.com/golangci/golint/cmd/golint@latest
	go install go.k6.io/xk6/cmd/xk6@latest

.PHONY: clean
clean:
	rm -f k6
	go clean -cache
