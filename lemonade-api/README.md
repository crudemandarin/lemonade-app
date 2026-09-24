# lemonade-api

A minimal Gin + GORM REST API on PostgreSQL, layered as controller → service → repository → model, with a `Sample` resource for CRUD.

## Dependencies

- [Go](https://go.dev/dl/) 1.27+ (modules are fetched automatically from `go.mod`)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for Postgres, and optionally the API)

## Project layout

```
main.go        entrypoint: loads secrets, connects DB, wires routes
libraries/     secrets loading, database connection
model/         GORM structs
repository/    DB access (CRUD queries)
service/       business logic / validation
controller/    HTTP handlers (Gin)
Dockerfile     API image (also used by the top-level docker-compose.yml)
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
curl http://localhost:8080/                          # health check
curl http://localhost:8080/samples                   # list
curl http://localhost:8080/samples/1                 # get one
curl -X POST http://localhost:8080/samples -H "Content-Type: application/json" -d '{"name":"hello"}'
curl -X PUT http://localhost:8080/samples/1 -H "Content-Type: application/json" -d '{"name":"updated"}'
curl -X DELETE http://localhost:8080/samples/1
```

## Format and lint

```bash
gofmt -w .           # format (VS Code does this on save)
golangci-lint run    # lint using .golangci.yml; install with: brew install golangci-lint
```

## API reference

| Method | Path           | Description      |
|--------|----------------|------------------|
| GET    | `/`            | Health check     |
| GET    | `/samples`     | List all samples |
| GET    | `/samples/:id` | Get one sample   |
| POST   | `/samples`     | Create a sample  |
| PUT    | `/samples/:id` | Update a sample  |
| DELETE | `/samples/:id` | Delete a sample  |

Request bodies are `{"name": "..."}`. Errors return `{"error": "..."}` with a 400 (`name is required`, `invalid request body`, `invalid id`), 404 (`sample not found`) or 500.

## Adding a new resource

Follow the `Sample` pattern:

1. `model/<name>.go`: GORM struct
2. `repository/<name>.go`: `New<Name>Repository(db)` + CRUD methods
3. `service/<name>.go`: `New<Name>Service(repository)` + business logic
4. `controller/<name>.go`: `New<Name>Controller(service)` + Gin handlers
5. `main.go`: construct repository → service → controller, then register routes
