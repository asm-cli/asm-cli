BINARY     := asm
MODULE     := github.com/asm-cli/asm-cli
VERPKG     := $(MODULE)/internal/version

VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X '$(VERPKG).Version=$(VERSION)' \
	-X '$(VERPKG).Commit=$(COMMIT)' \
	-X '$(VERPKG).BuildDate=$(BUILD_DATE)'

.PHONY: build install test clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	rm -f $(BINARY)
