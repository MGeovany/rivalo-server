// Atlas configuration for Rivalo migrations.
//
// Migrations target the Supabase Postgres over a session-mode connection
// (port 5432). Migration SQL is hand-authored because the schema references
// Supabase's managed `auth` schema (auth.users, auth.uid()), which a throwaway
// dev database cannot model; Atlas is used to version, apply, and track them.
//
// Usage (DATABASE_URL is read from the environment, e.g. via `make migrate-*`):
//   atlas migrate status --env supabase
//   atlas migrate apply  --env supabase

env "supabase" {
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://migrations"
  }
}
