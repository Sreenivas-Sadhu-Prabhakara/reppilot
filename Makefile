.PHONY: run build test fmt vet

run:
	go run ./cmd/server

build:
	mkdir -p bin
	go build -o bin/reppilot ./cmd/server

test:
	go test ./...

fmt:
	gofmt -l .

vet:
	go vet ./...
