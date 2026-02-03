.PHONY: build clean vet fmt tidy test proto

BINARY := ai
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/bluefunda/cai-cli/internal/cmd.Version=$(VERSION)"

build: tidy
	go build $(LDFLAGS) -o $(BINARY) ./cmd/ai

clean:
	rm -f $(BINARY)

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

test:
	go test -v -race -count=1 ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

proto:
	./scripts/generate-proto.sh
