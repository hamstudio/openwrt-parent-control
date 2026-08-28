.PHONY: all build test test-swift ipk clean deploy help

BINARY=bin/parentcontrold
VERSION ?= 1.0.0-1
ARCH ?= x86_64

all: build

build:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY) ./cmd/parentcontrold
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/relay-server ./cmd/relay-server

build-relay:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/relay-server ./cmd/relay-server

test:
	go test -v ./...

test-swift:
	cd client/ParentControlCore && swift test

ipk:
	chmod +x scripts/build_ipk.sh
	./scripts/build_ipk.sh $(ARCH) $(VERSION)

deploy: build
	chmod +x scripts/deploy.sh
	./scripts/deploy.sh

clean:
	rm -rf bin/ dist/

help:
	@echo "ParentControl Guard Makefile targets:"
	@echo "  make build       - Cross-compile parentcontrold and relay-server for linux/amd64"
	@echo "  make build-relay - Build standalone relay-server for current host"
	@echo "  make test        - Run Go backend unit tests"
	@echo "  make test-swift  - Run Swift Core cross-platform unit tests"
	@echo "  make ipk         - Build OpenWrt IPK package (Usage: make ipk ARCH=x86_64 VERSION=1.0.0-1)"
	@echo "  make deploy      - Build and deploy to target OpenWrt router via SCP/SSH"
	@echo "  make clean       - Remove built binaries and IPK distributions"
