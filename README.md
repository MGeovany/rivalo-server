# Rivalo: Server

Rivalo is built for amateur football. Each session captures what matters on the pitch: distance, sprints, heart rate, match context, opponent, result and hands it back to the app ready to summarize, compare, and improve week over week.

## Why it exists

- **Single source of truth**: profile, sessions, courts, goals, and stats live here; iOS and Watch capture and display.
- **Data that lasts**: PostgreSQL with explicit migrations; no magic in the schema.
- **Simple auth**: Supabase JWT; the app signs in, the server validates and responds.

## Stack

Go · PostgreSQL (Supabase) · Supabase Auth

## Project


| Repo            | What            |
| --------------- | --------------- |
| `rivalo-server` | API (this repo) |
| `rivalo-ios`    | iPhone          |
| `rivalo-watch`  | Apple Watch     |


## Status

Under construction, like the rest of Rivalo.