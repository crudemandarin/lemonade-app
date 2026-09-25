# Handoff: general implementor for small fixes

For a bot that takes small, self-contained fixes and tweaks (a bug, a copy change, a styling issue, a small missing test, a one-field API addition). Bigger features have their own handoffs; do not start those. Read this whole file, then `HANDOFF.md` (project overview and history), then `CLAUDE.md` (working rules) before your first fix.

## 1. The repo in one paragraph

A turn-based lemonade trading game. Go backend (`lemonade-api/`: Gin, GORM, Postgres, pure game rules in `internal/domain/`, thin handlers in `internal/api/`, storage in `internal/store/`) and an Angular 17 frontend (`lemonade-web/`: standalone components, signals, plain SVG charts, installable PWA). The **server is the source of truth**: the frontend renders server state and never computes game outcomes. `lemonade-web/src/app/core/api.models.ts` is the API contract. Deploy is Terraform plus scripts in `deploy/`.

## 2. Before every fix

1. `git status` and `git log --oneline | head`. **Other agents are editing this working tree at the same time**, often with a lot of uncommitted work in files you did not touch. Leave those files alone.
2. Reproduce the problem or read the code path. Find the smallest change that fixes it.
3. If the fix needs a design choice, a rule change, a balance change (prices, costs, upkeep, event odds, spreads), a schema change, or touches auth, stop and ask the user. Do not silently guess (this is a standing rule in `CLAUDE.md`).

## 3. How to do a fix

1. **Write a failing test first** when the bug is testable, then fix, then run the whole suite. Domain logic stays pure (no I/O, no clock, no framework imports); handlers and components stay thin; inject anything nondeterministic.
2. Match the surrounding code: naming, comment density, idiom. Prefer boring and readable. No speculative abstractions, no drive-by refactors, no extra features.
3. If you change a Go DTO, mirror it in `lemonade-web/src/app/core/api.models.ts` and `core/testing/fixtures.ts` in the same change.
4. UI conventions: sentence-case labels with no terminal punctuation, whole-dollar money everywhere, one primary button per screen, meaning never carried by colour alone, works at 390 px with no horizontal page scroll, and existing shared components (`shared/*`) reused before new ones are written.
5. Balance is guarded by tests with bands (`TestBalance*` in `internal/domain/balance_test.go`, plus the exploit-bot tests). If a fix changes an exact number asserted elsewhere (tests, `README.md`, `SPEC.md`), update them together and say so.

## 4. Commands

```bash
# backend (from lemonade-api/)
gofmt -l . ; go vet ./... ; go test ./...
DATABASE_URL="postgres://admin:password@localhost:5432/sample?sslmode=disable" go test ./internal/store   # real Postgres
BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v                                      # balance numbers

# frontend (from lemonade-web/)
npx ng test --watch=false --browsers=ChromeHeadless
npm run lint ; npm run format:check ; npm run build

# full stack (from repo root)
docker compose up -d --build     # web :4200, api :8080, db :5432; curl localhost:8080/api/health
```

Done means: gofmt clean, `go vet` clean, all Go tests pass, frontend tests, lint, format check and build pass, and for a UI fix you looked at it running (screenshot at 1280 and 390 px if you can). Say plainly if something was not run.

## 5. Gotchas

- **Angular 17 limits:** no `@let`; `as` aliasing only on the primary `@if` branch; use `afterNextRender`, not `ngAfterViewInit`, to measure widths. ESLint and Prettier deliberately ignore `design-preview/` and `shared/event-backdrop/`.
- **Docker compose** sometimes gets torn down by other sessions: if the API logs "no such host db", run `docker compose up -d` (the volume survives).
- **Health check** is `/api/health`, not `/healthz` (Cloud Run reserves that). nginx and the dev proxy pass `/api` through unchanged.
- **Persistence:** one game row per user with JSONB columns; `AutoMigrate` adds columns and old rows load with zero values. Any new persisted field must be safe for old saves. Every mutation is load, domain call, save in one transaction.
- **Determinism:** market prices come from `rand(seed ^ day)`; do not add unseeded randomness to game logic.
- The service worker is off in `ng serve`; PWA behavior needs a production build (compose's `web` container serves one).
- Auth is currently username-only via the `X-Username` header; a Firebase migration is planned (`HANDOFF-AUTH.md`). Do not touch auth in a small fix.

## 6. Git rules

- **Never `git add -A` or `git commit -a`.** Stage explicit paths of your own files only, and commit only those. If a file you must edit also has someone else's uncommitted changes, stop and tell the user rather than committing their work.
- One fix per commit. Small, imperative subject (for example `Fix sparkline tooltip clipping`). Do not push, amend, rebase, force anything, or touch other branches unless asked.
- If you deviated from the design or made a real trade-off, append an entry to `docs/claude/DECISIONS.md` (numbered after the last entry). Tick `PLAN.md` only if your fix completes a listed slice item.
- `deploy/**` and `docs/DEMO.md` are often being edited by someone else; do not modify them in a small fix.

## 7. Where the bigger work is documented (do not implement from these)

`HANDOFF-P0.md`, `HANDOFF-A-economy-and-runs.md`, `HANDOFF-B-scores.md`, `HANDOFF-C-glossary.md` (feature slices), `HANDOFF-BALANCE.md` (balance pass), `HANDOFF-AUTH.md` (Firebase sign-in). Some are already partly or fully implemented; check `PLAN.md` and `DECISIONS.md` for what has landed. If your fix overlaps one of them, mention it to the user instead of guessing.

## 8. When to stop and ask

A fix is not small if it: changes game rules or balance numbers, needs a schema or API shape change beyond one additive field, touches auth, deploy or infra, needs a new dependency, requires editing files with someone else's uncommitted changes, or you cannot reproduce or verify it. Report what you found, what you would change, and why, and wait.

## 9. Reporting

When you finish, say in a few lines: what was wrong, what you changed (with file links), which checks you ran and their results, anything you did not verify, and anything you noticed but deliberately left alone. If you find a "Log candidate" moment for `docs/claude/ai-usage-log.md` (a novel technique or a case where the process had to be corrected), flag it at the end and do not append it until the user confirms.
