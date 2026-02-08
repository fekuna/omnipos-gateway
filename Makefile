.PHONY: run build test clean tidy download help

# Default target
help:
	@echo "OmniPOS Gateway Service Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  run             - Run the gateway locally"
	@echo "  build           - Build the binary"
	@echo "  test            - Run tests"
	@echo "  tidy            - Tidy go modules"
	@echo "  download        - Download go modules"
	@echo "  clean           - Remove build artifacts"

run:
	go run cmd/http/main.go

build:
	go build -o bin/gateway cmd/http/main.go

test:
	go test -v -cover ./...

clean:
	rm -rf bin/

tidy:
	go mod tidy

download:
	go mod download
