.PHONY: build test run lint tidy docker-up docker-up-full docker-down proto demo demo-curls

APP_NAME=grounded-agent

build:
	go build -o bin/$(APP_NAME)$(shell go env GOEXE) ./cmd/server

test:
	go test ./...

run: build
	./bin/$(APP_NAME)$(shell go env GOEXE)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-up:
	docker compose up --build -d

docker-up-full:
	docker compose -f docker-compose.yml -f docker-compose.full.yml --profile full up --build -d

docker-down:
	docker compose --profile full down

# Offline sales/interview demo: mock MCP + demo LLM + mock retrieve (see docs/DEMO.md)
demo:
	@echo "Start mock MCP in another terminal: python scripts/mock_mcp_gateway.py"
	@echo "Then: LLM_MODE=demo RETRIEVE_MODE=mock MCP_GATEWAY_URL=http://127.0.0.1:8080 go run ./cmd/server"
	@echo "Curls: make demo-curls"

demo-curls:
	curl -s -X POST http://localhost:8000/agent/chat -H "Content-Type: application/json" \
	  -d "{\"query\":\"How many paid vacation days?\",\"session_id\":\"demo-kb\"}"; echo
	curl -s -X POST http://localhost:8000/agent/chat -H "Content-Type: application/json" \
	  -d "{\"query\":\"Create an HR ticket for leave approval\",\"session_id\":\"demo-ticket\"}"; echo
	curl -s -X POST http://localhost:8000/agent/chat -H "Content-Type: application/json" \
	  -d "{\"query\":\"What is the CEO salary on the Moon in 2099?\",\"session_id\":\"demo-oos\"}"; echo

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/retriever.proto
