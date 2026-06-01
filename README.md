# Rivalo — Server

API backend de Rivalo: cuentas, perfiles y sesiones deportivas para una app de seguimiento
físico de fútbol amateur.

## Stack

- Go (API REST/JSON)
- PostgreSQL administrado con Supabase
- Autenticación basada en JWT de Supabase Auth

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

## Estado

En desarrollo inicial.
