.PHONY: fmt tidy lint test install

default: fmt tidy lint test install

fmt:
	go fmt ./...

tidy:
	go mod tidy

lint:
	golangci-lint run

test:
	go test -v -covermode=count -coverprofile=coverage.out ./...

build:
	go build -ldflags '-s -w' -o conncheck ./cmd/conncheck

install:
	go install -ldflags '-s -w' ./cmd/conncheck
	conncheck -h 2>&1 | head -n1
