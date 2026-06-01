.PHONY: dev run test build tidy swagger

# Reload the server when .go files change (requires no other process on PORT).
dev:
	go run github.com/air-verse/air@v1.62.0

# One-shot run without file watching.
run:
	go run ./cmd/server

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server

tidy:
	go mod tidy

# Regenerate OpenAPI spec and docs package from swag annotations.
swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go --parseInternal --output docs
