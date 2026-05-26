.PHONY: build test install
BINARY := agent-notify

build:
	go build -o bin/$(BINARY) ./cmd/agent-notify

test:
	go test ./...

install: build
	install -m 755 bin/$(BINARY) $(HOME)/.local/bin/$(BINARY)
