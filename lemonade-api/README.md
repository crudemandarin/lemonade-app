# lemonade-api

The Gin + PostgreSQL API for Lemonade Tycoon. All game rules live in a pure domain package; the API layer loads a game, calls a domain function, and saves the result. See [DESIGN](../docs/claude/DESIGN.md).

## Dependencies

- [Go](https://go.dev/dl/) 1.27+ (modules are fetched automatically from `go.mod`)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for Postgres, and optionally the API)

## Project layout

```
main.go                 entrypoint: loads secrets, connects DB, migrates, wires routes
internal/domain/        pure game rules (no I/O, no Gin, no SQL): config (all balance knobs),
                        newgame, quotes, market, events, actions (buy/sell), facilities,
                        endday, bankruptcy, timeline; *_test.go beside each, plus balance_test.go
internal/store/         Repository interface, Postgres implementation, in-memory fake
internal/api/           Gin handlers, DTOs (camelCase JSON), error mapping, /api/health
libraries/              secrets loading, database connection
Dockerfile              API image (also used by the top-level docker-compose.yml)
```

## 1. Configure environment

```bash
cp .env.example .env
```

```
DB_HOST=127.0.0.1
DB_USERNAME=admin
DB_PASSWORD=password
DB_NAME=sample
DB_PORT=5432
```

`.env` is gitignored, so never commit real credentials.

## 2. Run the backend

Requires Postgres (step 3).

```bash
go run .    # reads .env, listens on :8080
```

### With Docker

```bash
docker build -t lemonade-api .
docker run --rm --name lemonade-api -p 8080:8080 \
  --env-file .env \
  -e DB_HOST=host.docker.internal \
  --add-host=host.docker.internal:host-gateway \
  lemonade-api
```

`127.0.0.1` inside a container means the container itself, so `DB_HOST=host.docker.internal` points the API at Postgres on your machine. `--add-host` is only needed on Linux.

## 3. Start Postgres

```bash
set -a; source .env; set +a
docker run -d --name lemonade-api-db \
  -e POSTGRES_USER="$DB_USERNAME" \
  -e POSTGRES_PASSWORD="$DB_PASSWORD" \
  -e POSTGRES_DB="$DB_NAME" \
  -p "$DB_PORT":5432 \
  -v lemonade-api-db-data:/var/lib/postgresql/data \
  postgres:16
```

```bash
docker stop lemonade-api-db    # stop
docker start lemonade-api-db   # start again
docker rm -f lemonade-api-db && docker volume rm lemonade-api-db-data   # delete container and data
```

Postgres reads the credentials only when it sets up an empty volume. To change the password later, delete the volume or run `ALTER USER`.

## 4. Try it out

```bash
curl http://localhost:8080/api/health
# Play on a username alone (the guest path):
curl -X POST http://localhost:8080/api/login -H "Content-Type: application/json" -d '{"username":"lemonjoe"}'
curl http://localhost:8080/api/game -H "X-Username: lemonjoe"
# A Google-secured account needs Authorization: Bearer <Firebase ID token> instead; locally, mint one from the emulator.
```

## Test

```bash
go test ./...
```

## Format and lint

```bash
gofmt -w .           # format (VS Code does this on save)
golangci-lint run    # lint using .golangci.yml; install with: brew install golangci-lint
```

## API reference

Every `/api/game`, `/api/scores` and `/api/runs` route identifies the player one of two ways. **Guest:** an `X-Username` header, accepted only for accounts that have not been secured (a secured one gets 401 `account_secured`). **Google:** `Authorization: Bearer <Firebase ID token>` (verified: signature, issuer, audience, expiry; provider `google.com`, verified email); a request with an Authorization header is judged by the token alone, never by falling back to the header. Missing or invalid: 401 `unauthorized`. A valid token whose account has no player yet: 403 `profile_required`. Google sign-in is optional and is off when `FIREBASE_PROJECT_ID` is unset. Every mutation returns the updated game view; errors return `{"error": "<code>", "message": "..."}`. The response shapes are defined in [api.models.ts](../lemonade-web/src/app/core/api.models.ts).

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/api/health` | Liveness check (no auth) |
| POST   | `/api/login` `{username}` | Guest login: create or get a user (a new user gets a new game). 409 `account_secured` if the name is secured with Google |
| GET    | `/api/me` | The caller's profile `{id, username}`, or 403 `profile_required` (token only, no player needed) |
| POST   | `/api/me/username` `{username}` | Create the player and their first game. 400 `invalid_username`, 409 `username_taken`, 409 `already_linked` |
| POST   | `/api/me/claim` `{username}` | Secure an existing username by linking the caller's Google account (this is also how a guest secures theirs). 404 `unknown_username`, 409 `already_claimed`, 409 `already_linked`, 429 `rate_limited` |
| GET    | `/api/game` | Current game view |
| POST   | `/api/game/new` | Start a fresh game |
| POST   | `/api/game/buy` `{resource, qty}` | Buy at ask |
| POST   | `/api/game/sell` `{resource, qty}` | Sell at bid |
| POST   | `/api/game/facilities/warehouse/expand` `{resource}` | Add a warehouse building for one resource |
| POST   | `/api/game/facilities/production/expand` | Add a production building |
| POST   | `/api/game/facilities/:type/upgrade` | Upgrade every building of `warehouse` or `production` |
| POST   | `/api/game/end-day` | End the day; returns the day report and the new game view |

The game view also carries `timeline` (capital and stock after each action, for the history charts) and `stats` (running totals for the end-of-game report). Old days are compacted to milestones to keep it small.

Game balance (prices, spread, events, tier costs) is one struct: `DefaultConfig()` in `internal/domain/config.go`.
