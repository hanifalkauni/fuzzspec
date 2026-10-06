.PHONY: all build test clean lint run-example validate

BINARY_CLI=fuzzspec.exe
BINARY_MCP=fuzzspec-mcp.exe

all: test build

build:
	go build -o $(BINARY_CLI) ./cmd/fuzzspec
	go build -o $(BINARY_MCP) ./cmd/fuzzspec-mcp

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

validate:
	./$(BINARY_CLI) validate --spec ./test/fixtures/petstore.yaml

clean:
	rm -f $(BINARY_CLI) $(BINARY_MCP) coverage.out coverage.html
