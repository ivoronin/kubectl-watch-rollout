.PHONY: build clean test install lint

build:
	go build -o bin/kubectl-watch_rollout ./cmd/kubectl-watch-rollout

clean:
	rm -rf bin/

test:
	go test ./...

install:
	go install ./cmd/kubectl-watch-rollout

lint:
	golangci-lint run ./...
