# FinTrack

Personal finance tracker — Go (Chi + pgx) + Postgres + Next.js. Track income/expenses,
accounts/wallets, **atomic transfers**, recurrences, monthly budgets, goals, reports,
CSV import/export with hash dedup, dark/light mode.

> Hard rule: money is **integer cents (`int64`/`BIGINT`)** across the whole backend.
> Zero floats.

## Quickstart (Docker)

```bash
cp .env.example .env
docker compose up --build
```

- API: http://localhost:8084 (`GET /healthz`, `GET /openapi.yaml`)
- Web: http://localhost:3004
- Postgres: localhost:5437 (`fintrack`/`fintrack`)

## Screenshots

Real screenshots against the local stack (`demo@fintrack.local`):

| Login | Dashboard |
|---|---|
| ![Login](docs/screenshots/01-login.png) | ![Dashboard](docs/screenshots/02-dashboard.png) |

| Transações | Contas |
|---|---|
| ![Transações](docs/screenshots/03-transactions.png) | ![Contas](docs/screenshots/04-accounts.png) |

| Categorias | Recorrências |
|---|---|
| ![Categorias](docs/screenshots/05-categories.png) | ![Recorrências](docs/screenshots/06-recurrences.png) |

| Orçamentos | Metas |
|---|---|
| ![Orçamentos](docs/screenshots/07-budgets.png) | ![Metas](docs/screenshots/08-goals.png) |

| Relatórios | Importar |
|---|---|
| ![Relatórios](docs/screenshots/09-reports.png) | ![Importar](docs/screenshots/10-import.png) |


`SEED_DEMO=true` (default in compose) seeds fictitious demo data on first boot
once an org exists (register via the web login page, then restart `api`).

## Local dev

```bash
# Backend (needs Postgres + JWT_SECRET)
export DATABASE_URL=postgres://fintrack:fintrack@localhost:5437/fintrack?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
go run ./cmd/api

# Frontend
cd web && npm install && npm run dev   # NEXT_PUBLIC_API_URL=http://localhost:8084
```

## API overview

Auth: `POST /api/v1/auth/register`, `POST /api/v1/auth/login` (returns
`{ user, orgs[], access_token }` + httpOnly refresh cookie),
`POST /api/v1/auth/refresh`, `GET /api/v1/me`. Frontend persists
`orgs[0].id` and sends `X-Org-ID`/`X-User`.

Resources: accounts, categories, transactions (`?from&to&kind&category_id&account_id`),
transfers (atomic), recurrences, budgets, goals, `reports/summary`, `reports/monthly`,
`import/csv` (dedup by sha256), `export/csv`. Full spec in `docs/openapi.yaml`.

## Tests

```bash
go build ./... && go vet ./... && go test ./...
cd web && npx tsc --noEmit && npm run build
```

Covers ledger math, 64-way concurrent transfers (sum conserved), CSV dedup,
auth roundtrips, plus integration tests (run with `DATABASE_URL` set).

## Env

See `.env.example`. Ports via `API_PORT` (8084), `WEB_PORT` (3004), `DB_PORT` (5437).

## License

MIT — see `LICENSE`.
