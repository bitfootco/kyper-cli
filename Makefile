BINARY    := kyper
PKG       := github.com/bitfootco/kyper-cli/internal/version
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -X $(PKG).Version=dev -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)

.PHONY: build install test vet lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kyper/

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/kyper/

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)
