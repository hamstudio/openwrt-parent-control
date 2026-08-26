.PHONY: build clean deploy

BINARY=bin/parentcontrold

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY) ./cmd/parentcontrold

clean:
	rm -rf bin/

deploy: build
	./scripts/deploy.sh
