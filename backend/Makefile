.PHONY: test run up down tidy

test:
	go test ./...

run:
	go run ./cmd/api

tidy:
	go mod tidy

up:
	docker compose up -d

down:
	docker compose down
