# lemonade-app

An Angular frontend, a Gin REST API and PostgreSQL, run together with Docker Compose.

| Service | Source | URL |
| ------- | ------ | --- |
| `web` | [lemonade-web/](lemonade-web/) (nginx) | http://localhost:4200 |
| `api` | [lemonade-api/](lemonade-api/) | http://localhost:8080 |
| `db`  | `postgres:16` | `localhost:${DB_PORT}` |

`web` forwards `/api/*` to `api` and removes the `/api` prefix, so no CORS setup is needed.

**Requires:** [Docker Desktop](https://www.docker.com/products/docker-desktop/)

## Run

```bash
cp .env.example .env    # first time: fill in DB credentials (see below)
docker compose up -d --build
```

```
DB_HOST=127.0.0.1
DB_USERNAME=admin
DB_PASSWORD=password
DB_NAME=sample
DB_PORT=5432
```

Inside Compose the API always connects to `db:5432`. `DB_HOST` only matters when you run the API outside Docker.

```bash
docker compose down           # stop
docker compose down -v        # stop and delete the database volume
docker compose logs -f api    # tail logs (also: web, db)
```

> Postgres reads `DB_PASSWORD` only when it creates a new volume. After changing it, run `docker compose down -v` (this deletes data) or `ALTER USER`.

## Try it

```bash
open http://localhost:4200                  # Angular app
curl http://localhost:4200/api/samples      # API via the web proxy
curl http://localhost:8080/samples          # API directly
```

For local development without Docker, see [lemonade-api](lemonade-api/README.md) and [lemonade-web](lemonade-web/README.md).

To deploy to Google Cloud (Cloud Run + Cloud SQL), see [deploy/](deploy/README.md).
