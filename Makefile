BINARY_NAME=forge
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test lint fmt vet clean install

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/forge

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

install: build
	cp $(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
	rm -rf public/
