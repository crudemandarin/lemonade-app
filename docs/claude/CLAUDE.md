# CLAUDE.md

Time-boxed (4h) greenfield take-home. Optimize for: working end-to-end product early, clean abstractions, tests, and being able to explain every decision.

## Stack & commands
- Full stack (from repo root): `cp .env.example .env && docker compose up -d --build` → web http://localhost:4200, api http://localhost:8080 (`/api/health`), db on `$DB_PORT`

- Language/runtime: Gin-Golang Backend (Go 1.27, in `lemonade-api/`)
- Install: `go mod download`
- Run: `docker compose up -d db` (from root), then `cp ../.env .env` (first time) and `go run .` (reads `lemonade-api/.env`, listens on :8080 or `$PORT`)
- Test: `go test ./...`
- Lint/format: `gofmt -w . && go vet ./...` (plus `golangci-lint run` if installed)

- Language/runtime: Angular-Typescript Frontend (Angular 17, Node 20+, in `lemonade-web/`)
- Install: `npm ci`
- Run: `npm start` (dev server at :4200, proxies `/api` to :8080)
- Test: `npx ng test --watch=false --browsers=ChromeHeadless`
- Lint/format: `npm run lint && npm run format`

## Start of every session
1. Read `SPEC.md`, `DESIGN.md`, `PLAN.md`, and `DECISIONS.md`.
2. Find the first unchecked slice in `PLAN.md`. Work on that slice only.
3. Tell me in 2-3 lines what you're about to do before doing it.

## Workflow per slice
1. Write failing tests that encode the slice's acceptance criteria.
2. Implement the minimum to pass.
3. Run the full test suite and the app. Both must work before moving on.
4. Commit (small, imperative message, e.g. `Add pricing calculation`).
5. Tick the slice off in `PLAN.md`.
6. If you deviated from `DESIGN.md` or made a real tradeoff, append an entry to `DECISIONS.md`.

## Architecture rules
- Domain logic is pure and isolated: no I/O, no UI, no framework imports.
- UI / CLI / API layers are thin and call into the domain.
- Inject anything nondeterministic (time, randomness) so tests are deterministic.
- Prefer boring, readable code over clever code. No speculative abstractions.

## Scope discipline
- Do not add features outside `SPEC.md`. Put ideas in the "Future work" section of `PLAN.md`.
- If a requirement is ambiguous, ask me; do not silently guess.
- Do not refactor unrelated code during a slice.

## Communication
- Explain design choices briefly as you make them. I must defend them in a walkthrough.
- Never leave the repo in a broken state at the end of a session.

## Rules of the exercise
- Repo must run from a fresh clone using only the README instructions.

## AI usage log
- Maintain `docs/ai-usage-log.md`. When I write a prompt (or you take an approach) that shows novel, unique, or advanced use of AI tooling, flag it at the end of your response with "Log candidate:" and a one-line reason.
- When I confirm, append a full entry using the template in that file. Include the verbatim prompt, why it works, the result, and the lesson.
- Also log moments where the AI got something wrong and the prompt or process was adjusted. Be selective: quality over quantity, target 5-10 strong entries total.
