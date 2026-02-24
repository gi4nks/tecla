BINARY_NAME=tecla
PREFIX ?= $(HOME)/.local/bin

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/tecla

run:
	go run ./cmd/tecla

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

release-snapshot:
	goreleaser release --snapshot --clean

install: build
	@mkdir -p $(PREFIX)
	@cp $(BINARY_NAME) $(PREFIX)/$(BINARY_NAME)

.PHONY: all build run clean test install
