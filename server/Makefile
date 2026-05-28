.PHONY: dev build test lint migrate openapi-gen clean

GO         := go
GOFLAGS    := -trimpath
LDFLAGS    := -s -w
BIN_DIR    := bin

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/server ./cmd/server
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/migrate ./cmd/migrate
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/tenantctl ./cmd/tenantctl

test:
	$(GO) test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

dev:
	$(GO) run ./cmd/server

migrate:
	$(GO) run ./cmd/migrate up

openapi-gen:
	@echo "TODO M3: regenerate frontend SDK from api/openapi.yaml"

clean:
	rm -rf $(BIN_DIR) coverage.out
