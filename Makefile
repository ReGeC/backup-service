APP := backup-service

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X backup-service/cmd.Version=$(VERSION) \
	-X backup-service/cmd.Commit=$(COMMIT) \
	-X backup-service/cmd.Date=$(DATE)

.PHONY: build release clean linux windows darwin darwin-arm test fmt

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP).exe .

linux:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o dist/$(APP)-linux-amd64 .

windows:
	@mkdir -p dist
	GOOS=windows GOARCH=amd64 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o dist/$(APP)-windows-amd64.exe .

darwin:
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o dist/$(APP)-darwin-amd64 .

darwin-arm:
	@mkdir -p dist
	GOOS=darwin GOARCH=arm64 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o dist/$(APP)-darwin-arm64 .

release: linux windows darwin darwin-arm

fmt:
	go fmt ./...

test:
	go test ./...

clean:
	rm -rf bin dist