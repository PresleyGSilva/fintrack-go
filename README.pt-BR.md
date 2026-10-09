# FinTrack

Controle financeiro pessoal — Go (Chi + pgx) + Postgres + Next.js. Receitas/despesas,
contas/carteiras, **transferências atômicas**, recorrências, orçamentos mensais,
metas, relatórios, importação/exportação CSV com dedup por hash, modo dark/light.

> Regra dura: dinheiro é **centavos inteiros (`int64`/`BIGINT`)** em todo o backend.
> Zero float.

## Início rápido (Docker)

```bash
cp .env.example .env
docker compose up --build
```

- API: http://localhost:8084 (`GET /healthz`, `GET /openapi.yaml`)
- Web: http://localhost:3004
- Postgres: localhost:5437 (`fintrack`/`fintrack`)

## Screenshots

Screenshots reais contra a stack local (`demo@fintrack.local`):

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


`SEED_DEMO=true` (padrão no compose) popula dados fictícios no primeiro boot
assim que uma org existir (cadastre-se na página de login e reinicie o `api`).

## Dev local

```bash
# Backend (precisa de Postgres + JWT_SECRET)
export DATABASE_URL=postgres://fintrack:fintrack@localhost:5437/fintrack?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
go run ./cmd/api

# Frontend
cd web && npm install && npm run dev   # NEXT_PUBLIC_API_URL=http://localhost:8084
```

## Visão da API

Auth: `POST /api/v1/auth/register`, `POST /api/v1/auth/login` (retorna
`{ user, orgs[], access_token }` + cookie httpOnly de refresh),
`POST /api/v1/auth/refresh`, `GET /api/v1/me`. O frontend persiste
`orgs[0].id` e envia `X-Org-ID`/`X-User`.

Recursos: contas, categorias, transações (`?from&to&kind&category_id&account_id`),
transferências (atômicas), recorrências, orçamentos, metas, `reports/summary`,
`reports/monthly`, `import/csv` (dedup por sha256), `export/csv`.
Especificação completa em `docs/openapi.yaml`.

## Testes

```bash
go build ./... && go vet ./... && go test ./...
cd web && npx tsc --noEmit && npm run build
```

Cobrem matemática do ledger, N transferências concorrentes (soma preservada),
dedup de CSV, roundtrips de auth, além de testes de integração (com `DATABASE_URL`).

## Env

Ver `.env.example`. Portas via `API_PORT` (8084), `WEB_PORT` (3004), `DB_PORT` (5437).

## Licença

MIT — ver `LICENSE`.
