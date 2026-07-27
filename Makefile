.PHONY: build test run lint tidy docker-up docker-up-full docker-down proto

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

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/retriever.proto
