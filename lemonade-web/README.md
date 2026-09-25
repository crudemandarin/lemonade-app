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
                   AuthService (Firebase SDK behind FirebaseAuthPort), SessionService
                   (guest name in localStorage; secured names in memory), bearer-token and 401/403 interceptors,
                   route guards, firebase-config (runtime), testing/{fixtures,fake-auth}.ts
  shared/          nav-bar, card, icon, money pipe, offline-banner, price-sparkline,
                   timeline-charts (capital/stock history), event-backdrop (CSS scenes)
  pages/home/      landing page ("Play game" / "Continue game")
  pages/signin/    username-only play ("Continue"), with optional "Sign in with Google"
  pages/username/  first Google sign-in: choose a username, or link an existing one
  pages/secure/    optional: link Google to the current guest username
  pages/game/      dashboard + game over; presentational components in components/:
                   stats-strip, events-banner, market-panel, facilities-panel,
                   day-report-modal, game-over
src/styles.scss    theme tokens (light/dark), buttons, inputs
ngsw-config.json   PWA service worker: caches the app shell, never /api
design-preview/    standalone event-backdrop preview (node design-preview/build.mjs)
```

Routes: `/` home, `/signin`, `/signin/username` (Google account with no player yet), `/secure` (guests only), `/game`, `/scores`, `/runs/:id` (guarded: nobody signed in goes to `/signin`; a Google account with no username goes to `/signin/username`).

Firebase settings are not built in: the app fetches `/config/firebase-config.json` at startup (`src/config/` for `ng serve`, rendered by nginx from `FIREBASE_*` env vars in the image). It is deliberately outside `/assets` so the service worker never caches it; `npm run build && npm run check:ngsw` verifies that.

It is an installable PWA (service worker in production builds only; the API is never cached, and actions are disabled offline). Only `GameStore` talks to `ApiService`. Page components read store signals and pass data down to presentational components, which emit events back up.

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
