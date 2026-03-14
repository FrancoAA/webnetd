BINARY  := webnetd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

GOBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || echo $(GOBIN)/golangci-lint)

.PHONY: all build test vet lint check clean install-lint

all: check build

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

install-lint:
	@if ! command -v golangci-lint >/dev/null 2>&1 && [ ! -f "$(GOBIN)/golangci-lint" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(GOBIN); \
	fi

lint: install-lint
	$(GOLANGCI_LINT) run

check: vet lint test

clean:
	rm -f $(BINARY)
