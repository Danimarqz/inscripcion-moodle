# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository layout

Three independent deployables that share one backend:

- `backend/` — Go 1.26 API (chi router, GORM, MariaDB/MySQL, Redis). Serves both the registration flow and the exam platform.
- `frontend-exam/` — Astro 6 + Preact + Tailwind v4 app for the exam platform (admin panel + public exam taking). Runs on port 4321.
- `frontend-inscripcion/` — Static vanilla JS/HTML enrollment form (served via nginx in Docker).

Production is deployed via `docker-compose.yml` using prebuilt GHCR images (`ghcr.io/danimarqz/backend-moodle`, `ghcr.io/danimarqz/frontend-exam`). The compose file expects an external `nginx_proxy` network and reads `backend/.env` (which is intentionally gitignored — do not commit a template).

Note: `ruff.toml` at the repo root is legacy from a prior Python backend and is not used by the current Go code.

## Common commands

### Backend (Go) — from `backend/`

```sh
go run ./cmd/api              # run API (reads backend/.env via godotenv)
go build ./cmd/api            # build
go test ./...                 # run all tests
go test ./internal/services/exam/...            # test one package
go test -run TestName ./internal/services/exam  # run a single test
go test -coverprofile=coverage.out -covermode=atomic ./...
golangci-lint run ./...       # CI uses v2.6
```

### Frontend-exam — from `frontend-exam/`

```sh
npm install
npm run dev        # astro dev on :4321
npm run build
npm run preview
```

### Frontend-inscripcion

Static files — serve `index.html` with any static server (nginx config is in `frontend-inscripcion/nginx.conf`).

## Backend architecture

Entry point: `backend/cmd/api/main.go` → `internal/server/server.go` wires everything. Read `server.go` first when orienting — it's the dependency graph in one file.

Layered layout under `backend/internal/`:

- `config/` — env loading (`backend/.env`). All runtime tuning lives here.
- `storage/` — MariaDB (GORM) and Redis clients. **AutoMigrate is intentionally disabled**; schema changes go through manual SQL migrations in `backend/db/`.
- `repository/` — GORM data access (exams, submissions). Controllers/services should go through repositories, not raw `*gorm.DB`, for anything new.
- `services/` — business logic, split by domain: `auth`, `email`, `exam`, `excelimport`, `moodle`, `admin`, `pdf`. Several services (`email`, `moodle`) expose package-level worker pools initialised once from `server.New` (`InitWorkerPool`, `InitSyncWorkerPool`) — don't re-init per request.
- `controllers/` — HTTP handlers grouped as `PublicController`, `RegisterController`, `AdminController`. Admin routes self-register via `adminController.RegisterRoutes(r)` on the `/admin` subrouter.
- `middleware/` — chi middleware plus a Redis-backed `RateLimiter` mounted globally (`cfg.RateLimitRequests` / `cfg.RateLimitWindow`).
- `cache/` — Redis cache layer for questions, public results, and admin failed-login tracking (brute-force protection).

Cross-cutting patterns to respect:

- **Redis is load-bearing**, not optional: rate limiting, caching of exam questions/public results, and admin login throttling all assume it's up. Code should fail closed if the client is nil.
- **Concurrency via worker pools + channels** (Go 1.26). Email sending, Moodle user sync, and Excel import all hand work off to goroutine pools — don't block request handlers on these.
- **Streaming Excel import** (`services/excelimport`) processes large XLSX files without loading them fully into memory; preserve the streaming approach when editing.
- **PDF generation** uses `gofpdf` and is tuned for low memory; generated files go to `backend/generated_pdfs/` (mounted as a Docker volume in prod).
- **Moodle integration** matches users fuzzily on name/surname/DNI to avoid duplicates — see `services/moodle`. Changes here are high-risk for data corruption.
- **Exam scoring** (penalizations, weighted scores, merits, passing thresholds) lives in `services/exam/calculation*.go` and has tests (`calculation_test.go`). This is the most churn-prone area — run those tests on any change.

## Frontend-exam architecture

Astro with Preact islands, Tailwind v4 via `@tailwindcss/vite`, SSR via `@astrojs/node`. Source under `frontend-exam/src/`:

- `pages/` — Astro routes (public exam pages + `/admin/*` panel).
- `components/` — Preact islands; the admin `ExamPage` was recently refactored into smaller modules (see recent commits) — prefer continuing that decomposition over adding to any monolith.
- `services/` — API client code talking to the Go backend.
- `hooks/`, `utils/`, `types/`, `constants/` — shared frontend code.

Admin results pages rely on **lazy-loaded student answers** — the list endpoint intentionally omits them and they are fetched on-demand when editing. Don't add them back to the list payload.

## Testing

Go tests live next to the code they cover. Notable suites:

- `internal/services/exam/calculation_test.go` — scoring/penalization math.
- `internal/services/admin/*_test.go` — admin service + exam update flow.
- `internal/services/excelimport/service_test.go` — streaming import.
- `internal/controllers/admin_test.go` — HTTP layer.

There is no frontend test suite.

## Project conventions (from memory / prior feedback)

- `backend/.env` being gitignored with no committed template is intentional — don't "fix" it by adding `.env.example`.
- Docker setup, lack of DB foreign keys, healthcheck omissions, and any `--legacy-peer-deps` usage are deliberate choices — don't flag them as issues or change them without being asked.
