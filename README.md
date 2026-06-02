# Rivalo — Server

API backend de Rivalo: cuentas, perfiles y sesiones deportivas para una app de seguimiento
físico de fútbol amateur.

## Stack

- Go (API REST/JSON)
- PostgreSQL administrado con Supabase
- Autenticación basada en JWT de Supabase Auth

## API docs (Swagger)

With the server running (`make dev`), open [http://localhost:8080/docs](http://localhost:8080/docs) (or `/docs/index.html`).

After changing handler annotations, regenerate the spec:

```bash
make swagger
```

## Migrations (Atlas)

Schema migrations are managed with [Atlas](https://atlasgo.io) (`brew install ariga/tap/atlas`).
Migration SQL lives in `migrations/` and is applied against the Supabase Postgres over the
session pooler (port 5432). `DATABASE_URL` is read from `.env`.

```bash
make migrate-status              # show whether the DB is up to date
make migrate                     # apply pending migrations
make migrate-new name=add_x      # create a new migration file (opens $EDITOR)
make migrate-hash                # recompute atlas.sum after editing a migration
```

Migration SQL is hand-authored: the schema references Supabase's managed `auth` schema
(`auth.users`, `auth.uid()`), so `atlas migrate diff` against a throwaway dev database is not used.

## Endpoints (inicial)

- `GET /health`
- `GET /v1/me`, `PUT /v1/me`
- `POST /v1/sessions`, `GET /v1/sessions`, `GET /v1/sessions/{id}`

## Repositorios del proyecto

- `rivalo-server` — API backend (este repo)
- `rivalo-ios` — app iPhone
- `rivalo-watch` — app Apple Watch

## Seed (demo user)

Creates (or reuses) auth user `marlongeo1999+mid@gmail.com` with password `Rivalo@123`, a midfielder
profile, and five fake watch sessions with heart-rate samples.

```bash
# In .env: DATABASE_URL, SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY (service role, not anon key)
make migrate   # if needed
make seed
```

SQL-only variant (user must already exist in Supabase Auth):

```bash
psql "$DATABASE_URL" -f scripts/seed_demo_user.sql
```

## Desarrollo

Requiere Go y un proyecto Supabase. Configuración por variables de entorno (ver `.env` local).

```bash
make dev    # servidor con recarga al guardar cambios en .go (Air)
make run    # una sola ejecución, sin watch
```

Tras cambiar anotaciones Swagger, corre `make swagger` y Air recargará en el siguiente guardado de un `.go`, o toca cualquier archivo en `internal/` / `cmd/` para forzar el rebuild.

## Estado

En desarrollo inicial.
