# Architecture — FinTrack

Personal finance tracker: Go modular monolith + Postgres + Next.js.

## Layout

- `cmd/api` — HTTP entrypoint: slog JSON logs, migrate on boot, optional
  `SEED_DEMO` seed, graceful shutdown (10s), CORS.
- `internal/config` — env (`PORT`, `DATABASE_URL`, `JWT_SECRET`, `SEED_DEMO`).
- `internal/db` — pgx pool + embedded migration runner (`migrations/*.sql`).
- `internal/auth` — bcrypt (cost 11), JWT HS256 access tokens (15 min),
  opaque refresh tokens (sha256 stored, httpOnly cookie, 7d, rotation),
  in-memory login rate limiter (10/min per IP).
- `internal/models` — request/response shapes. Money is **int64 cents**
  everywhere; no `float` in the backend.
- `internal/ledger` — pure money math (summary totals, transfer validation)
  kept DB-free so it is unit-testable, including a 64-way concurrency test.
- `internal/csvimport` — CSV parsing with decimal→cents conversion without
  floats, row hashing (sha256) for idempotent re-imports.
- `internal/handlers` — Chi router, auth middleware (Bearer + `X-Org-ID`
  tenant scoping), CRUDs, transfers, reports, CSV import/export.
- `internal/seed` — fictitious demo data (accounts, categories,
  transactions, budgets, goals).
- `migrations/` — versioned up-only SQL (`0001_init`, `0002_indexes`).
- `docs/openapi.yaml` — OpenAPI spec, served at `GET /openapi.yaml`.
- `web/` — Next.js 15 App Router + Tailwind. Auth persists
  `fintrack_org_id = orgs[0].id`; every API call sends `X-Org-ID`/`X-User`.

## Data model

`users` → `memberships` → `orgs` (tenant). Everything financial hangs off
an org: `accounts` (`balance_cents` BIGINT), `categories` (income|expense),
`transactions` (kind, `amount_cents > 0`, date, `import_hash`, `transfer_id`),
`recurrences`, `budgets` (category + month), `goals`, `refresh_tokens`.

## Atomic transfers

`POST /api/v1/transfers` runs in one DB transaction: locks both account
rows with `SELECT ... FOR UPDATE` in stable id order (deadlock-safe),
validates same org / distinct accounts / positive amount / sufficient funds,
writes two `transfer` legs sharing a `transfer_id`, updates both balances.
Concurrent-transfer tests assert the total is conserved.

## Money

- Backend: `BIGINT` cents + Go `int64`. No floats, no `NUMERIC` math.
- Frontend: formats `cents/100` only for display via `Intl.NumberFormat`.

## API envelope

Success: `{ "data": ... }`. Errors: `{ "error": { "code", "message" } }`
(`VALIDATION_ERROR`, `UNAUTHORIZED`, `NOT_FOUND`, `INSUFFICIENT_FUNDS`,
`CONFLICT`).

## Ports (defaults)

API `8084` → container `8080`; web `3004` → `3000`; Postgres host `5437` →
container `5432`. Override via `API_PORT` / `WEB_PORT` / `DB_PORT`.
