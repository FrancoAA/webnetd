BINARY  := webnetd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null)

.PHONY: all build test vet lint check clean install-lint

all: check build

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

install-lint:
ifndef GOLANGCI_LINT
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
endif

lint: install-lint
	golangci-lint run

check: vet lint test

clean:
	rm -f $(BINARY)
