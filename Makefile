.PHONY: build test install cross
BINARY := agent-notify
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/agent-notify

test:
	go test ./...

install: build
	install -m 755 bin/$(BINARY) $(HOME)/.local/bin/$(BINARY)

cross:
	@mkdir -p dist
	@for target in \
		"linux amd64" \
		"linux arm64" \
		"darwin amd64" \
		"darwin arm64"; do \
		set -- $$target; \
		out=dist/$(BINARY)-$(VERSION)_$$1_$$2; \
		echo "building $$out"; \
		GOOS=$$1 GOARCH=$$2 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" ./cmd/agent-notify; \
	done
