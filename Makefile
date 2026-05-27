.PHONY: proto build test lint docker-up docker-down generate-keys

PROTO_DIR  := proto
GEN_DIR    := gen
BINARY_DIR := bin

# ── Protobuf ───────────────────────────────────────────────────────────────
proto:
	protoc \
	  --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  $(PROTO_DIR)/metrics/metrics.proto \
	  $(PROTO_DIR)/notary/notary.proto

# ── Build ──────────────────────────────────────────────────────────────────
build:
	mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_DIR)/prober    ./cmd/prober
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_DIR)/consensus ./cmd/consensus
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_DIR)/notary    ./cmd/notary

# ── Test ───────────────────────────────────────────────────────────────────
test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# ── Lint ───────────────────────────────────────────────────────────────────
lint:
	golangci-lint run --timeout 3m ./...

# ── Docker ─────────────────────────────────────────────────────────────────
docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

# ── Dev utilities ──────────────────────────────────────────────────────────
generate-keys:
	go run ./tools/keygen/...

# ── Solidity ───────────────────────────────────────────────────────────────
contract-compile:
	cd contracts && forge build

contract-deploy-sepolia:
	cd contracts && forge script script/Deploy.s.sol --rpc-url sepolia --broadcast

contract-test:
	cd contracts && forge test -vvv
