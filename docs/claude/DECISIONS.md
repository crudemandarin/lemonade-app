# DECISIONS

Deviations from DESIGN.md and real tradeoffs made during the build. Newest last.

## 1. Keep the scaffold's `main.go` at the module root (slice 0)
- **Design said:** `cmd/server/` for the entrypoint; `internal/domain`, `internal/store`, `internal/api` for layers.
- **Did:** `main.go` stays at `lemonade-api/` root. New code goes in `internal/` as designed (`/healthz` is in `internal/api`). The scaffold's `controller/ service/ repository/ model/` sample code stays until the game's own layers replace it.
- **Why:** the Dockerfile (`go build .`), Cloud Build and deploy scripts all build the module root. Moving it costs a deploy-pipeline change for no behavior gain in a 4h box.
- **Trade-off:** mixed layout for a few slices (scaffold packages next to `internal/`). Removing the sample resource is a cleanup, not part of any slice.

## 2. `/healthz` is liveness only (slice 0)
- **Did:** `GET /healthz` returns `200 {"status":"ok"}` without pinging the database.
- **Why:** the server exits at startup if it cannot connect to Postgres, so a running process implies a DB connection was made. Simple, and it can't flap on a slow query.
- **Trade-off:** does not detect a DB that drops after startup. A readiness check with `db.Ping()` is a one-line change if needed.

## 3. Frontend built ahead of the backend; `api.models.ts` is the contract
- **Did:** at the user's request, built the whole frontend structure (routes, store, API client, every screen from the UX mocks) before the game backend exists. `lemonade-web/src/app/core/api.models.ts` defines every request and response shape; the Go DTOs must match it.
- **Why:** backend and deploy work can proceed in parallel; the UI is verified by component tests with fixtures (`core/testing/fixtures.ts`).
- **Trade-off:** the contract may shift when the domain is implemented; any change is made in `api.models.ts` first so the compiler flags every affected component.

## 4. API JSON uses camelCase; the view carries display-ready values
- **Did:** fields are camelCase (`previousPrice`, `upkeepPerDay`), not the scaffold's snake_case. The game view includes derived values the UI shows (tier names, bid/ask, capacities, expand and upgrade costs, `upgrade: null` at max level) so the frontend never recomputes rules.
- **Why:** matches TypeScript naming with no mapping layer, and keeps the "server is the source of truth" rule strict.
- **Trade-off:** Go structs need explicit `json:"camelCase"` tags; a slightly larger payload.

## 5. UI error and busy behavior
- **Did:** a failed action keeps the current view and shows the server's `message` in a dismissible alert (the next successful call clears it). Buy/Sell/Expand stay enabled per the mocks; Upgrade is disabled only at max level ("Max level"). End day is disabled while any request is in flight, so the day cannot end mid-trade.
- **Trade-off:** no client-side validation of qty, funds, or capacity; the server is the single place those rules live.
