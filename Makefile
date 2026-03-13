BINARY  := webnetd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build test vet lint check clean

all: check build

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint:
	golangci-lint run

check: vet lint test

clean:
	rm -f $(BINARY)
