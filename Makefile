OUTPUT_DIR = ./builds
GIT_COMMIT = $(shell git rev-parse HEAD 2>/dev/null | cut -c1-7 || echo "dev")
VERSION = 2.0.0-alpha.3
BUILD_TIME = $(shell date +"%Y-%m-%d_%H:%M")
BUILD_OPTIONS = -ldflags "-X main.Version=$(VERSION) -X main.CommitID=$(GIT_COMMIT) -X main.BuildTime='$(BUILD_TIME)'"

.PHONY: all gotty clean test build deps help bindata frontend

all: gotty

frontend:
	@echo "Building frontend JS..."
	cd js && npm run build

bindata:
	@echo "Generating bindata..."
	@go-bindata -o server/asset.go -pkg server resources/...

gotty: frontend bindata main.go
	@echo "Building gotty..."
	go build ${BUILD_OPTIONS} -o gotty
	@echo "Build complete: ./gotty"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f gotty
	@rm -rf ${OUTPUT_DIR}

test:
	@echo "Running tests..."
	go test -v ./...

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

help:
	@echo "Usage:"
	@echo "  make          - Build gotty (default)"
	@echo "  make gotty    - Build gotty binary"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make test      - Run tests"
	@echo "  make deps      - Download dependencies"
	@echo "  make help      - Show this help message"

# Install binary to $GOPATH/bin
install: gotty
	@echo "Installing gotty to $(shell go env GOPATH)/bin/"
	@mkdir -p $(shell go env GOPATH)/bin/
	@cp gotty $(shell go env GOPATH)/bin/

# Install binary to /usr/local/bin (macOS)
install-mac: gotty
	@echo "Installing gotty to /usr/local/bin/ (requires sudo)"
	@sudo cp gotty /usr/local/bin/

# Install binary to ~/.local/bin
install-local: gotty
	@echo "Installing gotty to ~/.local/bin/"
	@mkdir -p ~/.local/bin/
	@cp gotty ~/.local/bin/
