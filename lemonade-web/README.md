# lemonade-web

The Angular 17 frontend for Lemonade Tycoon. It renders server state only: every game rule runs in [lemonade-api](../lemonade-api/), and every mutation returns the updated game view.

The app calls `/api/...` on its own origin. The dev server (`proxy.conf.json`) and nginx (`nginx.conf.template`) forward those calls to the API unchanged (the API serves its routes under `/api`).

## Dependencies

- [Node.js](https://nodejs.org/) 20 LTS or newer (Angular 17 supports 18.13+ and 20.9+)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/), to run the API or the full stack

## Project layout

```
src/app/
  core/            API contract (api.models.ts), ApiService, GameStore (signals),
                   SessionService, X-Username and 401 interceptors, auth guard
  shared/          nav-bar, card, icon, money pipe
  pages/home/      landing page ("Play game" / "Continue game")
  pages/signin/    username-only sign in
  pages/game/      dashboard + game over; presentational components in components/:
                   stats-strip, events-banner, market-panel, facilities-panel,
                   day-report-modal, game-over
src/styles.scss    theme tokens (light/dark), buttons, inputs
```

Routes: `/` home, `/signin`, `/game` (guarded: redirects to `/signin` without a stored username).

Only `GameStore` talks to `ApiService`. Page components read store signals and pass data down to presentational components, which emit events back up.

## Run locally

Start the API on `localhost:8080` first, with either the [lemonade-api](../lemonade-api/README.md) steps or `docker compose up -d db api` from the parent folder. Then:

```bash
npm ci       # install
npm start    # dev server at http://localhost:4200, with the /api proxy
```

```bash
npm run build                                          # production build
npx ng test --watch=false --browsers=ChromeHeadless    # unit tests
npm run format                                         # format with Prettier (format:check to verify)
npm run lint                                           # lint with angular-eslint
```

## Run with Docker

nginx forwards `/api` to a container named `api`, so run the image through the parent folder's Compose file:

```bash
cd .. && docker compose up -d --build        # full stack
docker compose up -d --build web             # rebuild only the frontend
```
