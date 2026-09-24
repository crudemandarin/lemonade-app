# lemonade-web

An Angular 17 single-page demo of the [lemonade-api](../lemonade-api/) API that uses every endpoint:

- **API status**: `GET /` on page load
- **Create sample**: `POST /samples`
- **Samples table**: `GET /samples`, inline edit (`PUT /samples/:id`), delete (`DELETE /samples/:id`)
- **Look up by ID**: `GET /samples/:id`
- **Request history**: every call with its method, status, timing and bodies

The app calls `/api/...` on its own origin. The dev server (`proxy.conf.json`) and nginx (`nginx.conf`) forward those calls to the API and remove the `/api` prefix.

## Dependencies

- [Node.js](https://nodejs.org/) 20 LTS (Angular 17 supports 18.13+ and 20.9+)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/), to run the API or the full stack

## Project layout

```
src/app/
  models/          API resource types
  services/        HTTP client, sample list store, request log + interceptor
  shared/          reusable UI components (card, icon, json-view, method-badge, status-pill)
  utilities/       helper functions (API error messages)
  pages/index/     the demo page and its section components
src/styles.scss    theme (light/dark), buttons, inputs
```

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
npm run lint -- --fix                                  # lint and apply safe auto-fixes
```

## Run with Docker

nginx forwards `/api` to a container named `api`, so run the image through the parent folder's Compose file:

```bash
cd .. && docker compose up -d --build        # full stack
docker compose up -d --build web             # rebuild only the frontend
```

Run on its own with `docker run`, the app loads but every API call fails.
