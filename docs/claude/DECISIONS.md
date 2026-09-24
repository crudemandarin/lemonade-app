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
