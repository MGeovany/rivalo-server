.PHONY: dev run test build tidy swagger migrate-status migrate migrate-new migrate-hash seed upload-player-card-assets

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

# --- Database migrations (Atlas) --------------------------------------------
# DATABASE_URL is read from .env and must use the session-mode connection
# (Supabase session pooler, port 5432).

# Show whether the database is up to date with the migration files.
migrate-status:
	@DATABASE_URL="$$(grep '^DATABASE_URL=' .env | cut -d= -f2-)" atlas migrate status --env supabase

# Apply all pending migrations to the database.
migrate:
	@DATABASE_URL="$$(grep '^DATABASE_URL=' .env | cut -d= -f2-)" atlas migrate apply --env supabase

# Create a new, empty migration file: make migrate-new name=add_sessions
migrate-new:
	@test -n "$(name)" || (echo "usage: make migrate-new name=<description>" && exit 1)
	atlas migrate new $(name) --dir file://migrations --edit

# Recompute the migration directory checksum (run after editing a migration file).
migrate-hash:
	atlas migrate hash --dir file://migrations

# Demo user marlongeo1999+mid@gmail.com / Rivalo@123 + 5 fake watch sessions.
# Requires DATABASE_URL, SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY in .env.
seed:
	go run ./cmd/seed

# Upload layered player card PNGs to Supabase Storage (requires migration applied).
upload-player-card-assets:
	@bash scripts/upload-player-card-assets.sh
