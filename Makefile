PROJECT := kubectl-watch-rollout
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test test-all lint release clean install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/kubectl-watch_rollout ./cmd/$(PROJECT)

test:
	go test -race ./...

test-all: lint test
	@echo "All tests passed"

lint:
	golangci-lint run

release:
	goreleaser release --clean

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/$(PROJECT)

clean:
	rm -rf bin/ dist/
