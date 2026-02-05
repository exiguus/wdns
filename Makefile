.PHONY: all lint test build docker run

# Default: run lint, tests and build
all: lint test build

lint:
	golangci-lint run ./...

test:
	go test ./... -v

# Build the local binary for quick local runs
build:
	[ -d bin ] || mkdir bin
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) \
		go build -trimpath -ldflags="-s -w" -o bin/wdns ./cmd/wdns

# Build the Docker image (uses Dockerfile at project root)
docker-build:
	# Build a local image for the host platform.
	docker build -t wdns:local .

# Run docker-compose for local testing (builds images as needed)
docker-run:
	docker compose up --build

# Run the locally-built binary (builds it first if needed)
run: build
	./bin/wdns
