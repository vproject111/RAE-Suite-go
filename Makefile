.PHONY: all build test clean build-cross audit-5stage verify-parity

BIN_DIR := bin
MODULES := rae-memory rae-crl rae-mcp rae-suite rae-quality rae-lab rae-phoenix rae-hive rae-portal rae

all: test build

build:
	@mkdir -p $(BIN_DIR)
	@for mod in $(MODULES); do \
		echo "Building $(BIN_DIR)/$$mod (CGO_ENABLED=0)..."; \
		CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$$mod ./cmd/$$mod; \
	done

test:
	CGO_ENABLED=0 go vet ./...
	CGO_ENABLED=0 go test -v ./...
	go test -race ./...

build-cross:
	@mkdir -p $(BIN_DIR)/cross
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/cross/rae-linux-amd64 ./cmd/rae
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/cross/rae-linux-arm64 ./cmd/rae
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/cross/rae-windows-amd64.exe ./cmd/rae
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/cross/rae-darwin-arm64 ./cmd/rae

audit-5stage: build
	python3 scripts/reusable_5stage_parity_auditor.py

verify-parity: build
	python3 scripts/verify_python_vs_go_parity.py

clean:
	rm -rf $(BIN_DIR)
