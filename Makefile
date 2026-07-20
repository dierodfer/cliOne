BINARY := clione
BIN_DIR := bin

.PHONY: build test lint vet run clean

build:
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/clione

test:
	go test ./...

vet:
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; running go vet instead"; \
		go vet ./...; \
	fi

run:
	go run ./cmd/clione

clean:
	rm -rf $(BIN_DIR) dist coverage.out
