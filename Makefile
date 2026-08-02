APP_NAME := localbridge
MODULE := github.com/XyUFlaW1eSs/LocalBridge

.PHONY: all build test vet fmt run clean

all: test build

build:
	go build -trimpath -ldflags "-s -w" -o bin/$(APP_NAME) ./cmd/localbridge

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

run:
	go run ./cmd/localbridge -config configs/config.example.yaml

clean:
	go clean
