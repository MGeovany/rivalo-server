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

## Endpoints (inicial)

- `GET /health`
- `GET /v1/me`, `PUT /v1/me`
- `POST /v1/sessions`, `GET /v1/sessions`, `GET /v1/sessions/{id}`

## Repositorios del proyecto

- `rivalo-server` — API backend (este repo)
- `rivalo-ios` — app iPhone
- `rivalo-watch` — app Apple Watch

## Desarrollo

Requiere Go y un proyecto Supabase. Configuración por variables de entorno (ver `.env` local).

```bash
make dev    # servidor con recarga al guardar cambios en .go (Air)
make run    # una sola ejecución, sin watch
```

Tras cambiar anotaciones Swagger, corre `make swagger` y Air recargará en el siguiente guardado de un `.go`, o toca cualquier archivo en `internal/` / `cmd/` para forzar el rebuild.

## Estado

En desarrollo inicial.
