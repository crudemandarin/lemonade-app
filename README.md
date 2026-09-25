<p align="center">
  <img src="lemonade-web/src/assets/ui/logo.svg" width="96" alt="Lemonade Tycoon logo">
</p>

<h1 align="center">Lemonade Tycoon</h1>

<p align="center">
  <em>Buy low, squeeze smart, sell high, and survive the market one day at a time.</em>
</p>

<p align="center">
  <img alt="Angular 17" src="https://img.shields.io/badge/Angular-17-dd0031?logo=angular&logoColor=white">
  <img alt="Go 1.27" src="https://img.shields.io/badge/Go-1.27-00add8?logo=go&logoColor=white">
  <img alt="Gin" src="https://img.shields.io/badge/API-Gin-008ecf">
  <img alt="PostgreSQL 16" src="https://img.shields.io/badge/PostgreSQL-16-4169e1?logo=postgresql&logoColor=white">
  <img alt="Google Cloud Run" src="https://img.shields.io/badge/Cloud%20Run-deployed-4285f4?logo=googlecloud&logoColor=white">
</p>

<p align="center">
  <img src="lemonade-web/src/assets/resources/lemon.svg" width="48" alt="Lemon">
  <img src="lemonade-web/src/assets/resources/sugar.svg" width="48" alt="Sugar">
  <img src="lemonade-web/src/assets/resources/ice.svg" width="48" alt="Ice">
  <img src="lemonade-web/src/assets/resources/cup.svg" width="48" alt="Cup">
  <img src="lemonade-web/src/assets/ui/end-day.svg" width="32" alt="becomes">
  <img src="lemonade-web/src/assets/resources/lemonade.svg" width="48" alt="Lemonade">
</p>

A turn-based lemonade business game. Buy ingredients, grow your facilities, sell lemonade, and try not to go bankrupt. Angular frontend, Gin (Go) REST API, PostgreSQL.

> **Design:** the technical design doc, [docs/numeric-tdd.md](docs/numeric-tdd.md), covers architecture, game rules, data model, API, balance and tuning, and trade-offs. Start there.

## The game in 30 seconds

1 lemon + 1 sugar + 1 ice + 1 cup = 1 lemonade. Nothing happens until you press **End day**, so take your time.

| | Step | What happens |
| :-: | ---- | ------------ |
| <img src="lemonade-web/src/assets/ui/coin.svg" width="24" alt=""> | **Trade** | Prices wander every day. Buy at the ask, sell at the bid, and mind the 10% spread. |
| <img src="lemonade-web/src/assets/ui/factory.svg" width="24" alt=""> | **Produce** | At end of day your production facility turns ingredients into lemonade. |
| <img src="lemonade-web/src/assets/ui/event.svg" width="24" alt=""> | **Ride the events** | Heat waves, rainy weeks, lemon blights and more push prices around for a few days. |
| <img src="lemonade-web/src/assets/ui/upgrade.svg" width="24" alt=""> | **Grow** | Expand and upgrade warehouses and production, but every building costs daily upkeep. |
| <img src="lemonade-web/src/assets/ui/sad-face.svg" width="24" alt=""> | **Survive** | If you cannot pay upkeep, stock is sold off to cover it. If that is not enough, the run ends. |

<p align="center">
  <img src="lemonade-web/src/assets/facilities/warehouse-1.svg" width="56" alt="Pantry">
  <img src="lemonade-web/src/assets/facilities/warehouse-2.svg" width="56" alt="Garage">
  <img src="lemonade-web/src/assets/facilities/warehouse-3.svg" width="56" alt="Barn">
  <img src="lemonade-web/src/assets/facilities/warehouse-4.svg" width="56" alt="Industrial warehouse">
  &nbsp;&nbsp;
  <img src="lemonade-web/src/assets/facilities/production-1.svg" width="56" alt="Kitchen">
  <img src="lemonade-web/src/assets/facilities/production-2.svg" width="56" alt="Food truck">
  <img src="lemonade-web/src/assets/facilities/production-3.svg" width="56" alt="Bottling plant">
  <img src="lemonade-web/src/assets/facilities/production-4.svg" width="56" alt="Lemonade factory">
</p>

The game has a built-in **How to play** menu in the top bar, with a quick start and a glossary.

## Highlights

- **Server is the source of truth.** The UI renders server state; every action returns the full game view.
- **Pure, deterministic domain.** All rules live in `internal/domain` with no I/O, and randomness is seeded, so the same seed and actions give the same game.
- **Safe concurrency.** Every action is `load, domain call, save` in one transaction with a row lock.
- **Play as a guest or with Google.** A username is enough to play; Google sign-in secures the account.
- **Installable PWA** with an offline banner, scores board, and per-run history.
- **One-command deploy** to Google Cloud with Terraform and keyless GitHub Actions.

## Architecture

```mermaid
flowchart LR
  user([Player's browser<br/>Angular PWA])
  dns[Cloudflare DNS<br/>lemonade.nyko.run]

  subgraph gcp[Google Cloud project]
    web[Cloud Run: web<br/>nginx serves SPA,<br/>proxies /api/*]
    api[Cloud Run: api<br/>Go + Gin]
    sql[(Cloud SQL<br/>Postgres 16)]
    sm[Secret Manager<br/>db-password]
    ar[Artifact Registry<br/>images]
    cb[Cloud Build]
  end

  gh[GitHub Actions<br/>push to main]

  user -->|HTTPS| dns --> web
  web -->|/api/* HTTPS| api
  api -->|/cloudsql unix socket| sql
  sm -.->|secret at startup| api
  gh -->|keyless login via<br/>Workload Identity| cb
  cb -->|push images| ar
  ar -.->|deploy new revision| web
  ar -.->|deploy new revision| api
```

The browser only ever talks to `web`; its nginx forwards `/api/*` to `api`, so there is no CORS. Locally, Docker Compose runs the same three pieces.

Inside the API, dependencies point inward:

```mermaid
flowchart LR
  http[internal/api<br/>Gin handlers, DTOs, auth] --> domain[internal/domain<br/>pure game rules]
  http --> store[internal/store<br/>Repository interface]
  store --> pg[(Postgres)]
  store -.->|in-memory fake for tests| domain
```

More detail is in [the technical design doc](docs/numeric-tdd.md#2-architecture).

## Quick start

```bash
cp .env.example .env
docker compose up -d --build
open http://localhost:4200
```

Full instructions (auth emulator, hot reload, tests) are further down.

| Service | Source | URL |
| ------- | ------ | --- |
| `web` | [lemonade-web/](lemonade-web/) (nginx) | http://localhost:4200 |
| `api` | [lemonade-api/](lemonade-api/) | http://localhost:8080 |
| `db`  | `postgres:16` | `localhost:${DB_PORT}` |

`web` forwards `/api/*` to `api` unchanged (the API serves its routes under `/api`), so no CORS setup is needed.

Working docs (in [docs/claude/](docs/claude/)): [SPEC](docs/claude/SPEC.md) (rules), [DESIGN](docs/claude/DESIGN.md) (architecture), [PLAN](docs/claude/PLAN.md) (slices), [DECISIONS](docs/claude/DECISIONS.md) (deviations and trade-offs), [ai-usage-log](docs/claude/ai-usage-log.md).

## Code structure

```
lemonade-api/    Go + Gin REST API. internal/domain holds every game rule (pure, no I/O);
                 internal/store is Postgres persistence; internal/api is thin HTTP handlers
lemonade-web/    Angular 17 SPA + PWA. Renders server state only; GameStore is the one API client
deploy/          Terraform + scripts for Google Cloud (Cloud Run, Cloud SQL)
docs/            spec, design, decisions, plan, AI usage log
docker-compose.yml   db + api + web for local use
.github/         deploy-on-push workflow
```

Each codebase has its own README with layout and commands: [lemonade-api](lemonade-api/README.md), [lemonade-web](lemonade-web/README.md), [deploy](deploy/README.md).

## Requirements

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (running), to start the full stack
- For local development and tests: [Go](https://go.dev/dl/) 1.27+ and [Node.js](https://nodejs.org/) 20+ with Chrome (for headless frontend tests)

## Install

```bash
cp .env.example .env                        # DB credentials (defaults work locally)
cd lemonade-api && go mod download && cd ..
cd lemonade-web && npm ci && cd ..
```

`.env.example` contains:

```
DB_HOST=127.0.0.1
DB_USERNAME=admin
DB_PASSWORD=password
DB_NAME=sample
DB_PORT=5432

# Optional Google sign-in (secures an account); players can always play on a username alone.
FIREBASE_PROJECT_ID=demo-lemonade
# Local Firebase Auth Emulator (docker compose --profile auth up); leave unset in production.
FIREBASE_AUTH_EMULATOR_HOST=localhost:9099
```

## Run

### Full stack (Docker Compose)

```bash
docker compose --profile auth up -d --build   # the "auth" profile adds the Firebase Auth Emulator
open http://localhost:4200                  # the game; "Continue with Google" opens the emulator's fake Google
curl http://localhost:4200/api/health      # API health via the web proxy: {"status":"ok"}
```

Sign-in runs against the [Firebase Auth Emulator](https://firebase.google.com/docs/emulator-suite/connect_auth) (no real Google account, no Firebase project needed): in the popup choose "Add new account", "Auto-generate user information", then "Sign in with Google.com". Its account list is at http://localhost:4000/auth. Without `--profile auth` you can still play on a username alone; only "Sign in with Google" and linking need the emulator. The emulator forgets its accounts when it stops.

```bash
docker compose down           # stop
docker compose down -v        # stop and delete the database volume
docker compose logs -f api    # tail logs (also: web, db)
```

> Postgres reads `DB_PASSWORD` only when it creates a new volume. After changing it, run `docker compose down -v` (this deletes data) or `ALTER USER`.

### Local development (hot reload)

```bash
docker compose up -d db                                  # database only
docker compose --profile auth up -d auth-emulator        # Firebase Auth Emulator on :9099 (UI on :4000)
(cd lemonade-api && cp ../.env .env && go run .)         # API on :8080 (keeps running; use a new terminal for the next line)
(cd lemonade-web && npm start)                           # web on :4200, proxies /api to :8080
```

`npm start` reads `lemonade-web/src/config/firebase-config.json`, which points at the emulator (without the emulator running, only username play works). To try the API by hand: `curl -X POST localhost:8080/api/login -H 'Content-Type: application/json' -d '{"username":"lemonjoe"}'`, then send `-H 'X-Username: lemonjoe'`.

Inside Compose the API always connects to `db:5432`. `DB_HOST` only matters when you run the API outside Docker.

## Test

```bash
(cd lemonade-api && go test ./...)
(cd lemonade-web && npx ng test --watch=false --browsers=ChromeHeadless)
```

Optional integration test of the token verifier against the emulator (skipped otherwise): start it with `docker compose --profile auth up -d auth-emulator`, then `FIREBASE_AUTH_EMULATOR_HOST=localhost:9099 go test ./internal/auth`.

Lint and format: `gofmt -w . && go vet ./...` in `lemonade-api/`; `npm run lint && npm run format` in `lemonade-web/`.

## Sign in

**Anyone can play on a username alone**: type a name, start a game. That is the brief's "no password" login, and it is not secure: whoever types a name plays as that player. **Signing in with Google is optional and secures an account.** A guest chooses "Secure account", links a Google account, and from then on only that Google account can play the username (a bare username is refused with `account_secured`). A player can also start with Google, choosing a username on first sign-in, or link a Google account to a username they already had. The web app sends the Firebase ID token as `Authorization: Bearer <token>`; the API verifies it (Google provider, verified email) and maps the account's Firebase UID to a player. **The username is the only public identity**: the Google name and email are never shown or returned by any endpoint (the email is stored for the account and nothing reads it back).

Setup for a real GCP project is a one-time runbook in [deploy/README.md](deploy/README.md#sign-in-with-google-firebase-auth). Locally it needs nothing but the emulator (see Run above). With no `FIREBASE_PROJECT_ID` the API runs guests-only and the app hides the Google buttons.

## Deploy

Google Cloud (Cloud Run + Cloud SQL), see [deploy/](deploy/README.md). Pushes to `main` redeploy automatically. Google sign-in needs the one-time Firebase setup (see Sign in); username play works without it.

## Tuning the game

All balance numbers (prices, volatility, events, facility costs and upkeep) are one struct: `DefaultConfig()` in [lemonade-api/internal/domain/config.go](lemonade-api/internal/domain/config.go). Change a value and restart the API (`docker compose up -d --build api`); the frontend needs no changes. The values, what each knob does and the balance analysis are in the [technical design doc, section 9](docs/numeric-tdd.md#9-balance-and-tuning).

```bash
cd lemonade-api
go test ./internal/domain                                            # balance guard-rail tests
BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v # full report
```

Changing a price, cost or upkeep also changes a few exact numbers asserted in the API and domain tests (for example the $775 first warehouse upgrade); update those alongside.

## Known limitations

- **Username-only play is not secure**, by design of the brief: anyone can type another guest's name and play (or read their run history) as them. Securing an account with Google is the fix, and it is optional.
- **Securing is first come, first served.** Nothing proves a guest owns their username, so whoever links Google to a name first owns it; a hostile player could secure someone else's guest name before they do. Claims are rate limited (5 attempts per 10 minutes per Google account, in memory, so per API instance). Possible hardening, not built: a claim deadline after which unclaimed legacy accounts are archived.
- Usernames are public on the global high score board (3 to 40 ASCII characters, not case sensitive: `Joe` and `joe` are the same player). Nothing else about a player is public. A player's run detail is readable only by that player (for a guest, by whoever types the name).
- Tokens live up to an hour and are checked locally, not for revocation, so disabling a Google account takes effect within the hour.
- One Google account per player; no account deletion, no username change, no un-securing, no other sign-in providers.
