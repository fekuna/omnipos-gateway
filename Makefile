.PHONY: run build test clean

# ==============================================================================
# Development

run:
	go run cmd/http/main.go

build:
	go build -o bin/gateway cmd/http/main.go

test:
	go test -v -cover ./...

clean:
	rm -rf bin/

# ==============================================================================
# Dependencies

tidy:
	go mod tidy

download:
	go mod download
