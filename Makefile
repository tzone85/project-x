BINARY  := px
PKG     := github.com/tzone85/project-x
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint vet clean install

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/px

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

vet:
	go vet ./...

clean:
	rm -rf bin/ coverage.out

install:
	go install $(LDFLAGS) ./cmd/px
