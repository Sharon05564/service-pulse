# ServicePulse

A Go-based API uptime monitoring and incident tracking platform.

ServicePulse lets you register any API or website, then continuously checks it
on a configurable interval. Every result is stored in PostgreSQL. Outages
automatically open incidents; recoveries resolve them. A summary endpoint
exposes uptime percentage, average latency, and more.

---

## Tech Choices

| Concern | Choice | Why |
|---|---|---|
| Router | [Chi](https://github.com/go-chi/chi) | Lightweight, idiomatic, no magic — close to the standard library |
| Database access | [sqlx](https://github.com/jmoiron/sqlx) | You write real SQL (great for learning), struct scanning, no code generation step |
| Migrations | Plain SQL files | Simple, portable, no extra tooling |
| Logging | `log/slog` (stdlib) | Structured JSON logging built into Go 1.21+, zero dependencies |

---

## Architecture

```
cmd/
  api/
    main.go           # Entry point: wires everything together, starts server + worker

internal/
  config/             # Loads env vars / .env file
  database/           # PostgreSQL connection + migration runner
  models/             # Struct definitions for Service, CheckResult, Incident + DTOs
  repository/         # Database layer — one file per domain, interface-driven
  handlers/           # HTTP handlers + chi router
  worker/             # Background health checker goroutine

migrations/           # Numbered SQL migration files (run on startup)
```

The background worker and the HTTP server run concurrently. The worker loads
active services every 10 seconds, then fans out a goroutine per service that is
due for a check. Results are written to `check_results`; incidents are opened /
resolved automatically.

---

## API Endpoints

### System

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Returns `{"status":"ok"}` |

### Services

| Method | Path | Description |
|---|---|---|
| POST | `/api/services` | Register a new service to monitor |
| GET | `/api/services` | List all services |
| GET | `/api/services/{id}` | Get one service |
| PUT | `/api/services/{id}` | Update a service |
| DELETE | `/api/services/{id}` | Delete a service |

### Health Check Results

| Method | Path | Description |
|---|---|---|
| GET | `/api/services/{id}/checks` | Recent check results (`?limit=50`) |

### Incidents

| Method | Path | Description |
|---|---|---|
| GET | `/api/services/{id}/incidents` | All incidents for a service |

### Metrics

| Method | Path | Description |
|---|---|---|
| GET | `/api/services/{id}/summary` | Uptime %, avg latency, total checks, active incident |

---

## Data Models

**Service**

```json
{
  "id": "uuid",
  "name": "GitHub API",
  "url": "https://api.github.com",
  "method": "GET",
  "expected_status_code": 200,
  "check_interval_seconds": 60,
  "is_active": true,
  "created_at": "...",
  "updated_at": "..."
}
```

**CheckResult**

```json
{
  "id": "uuid",
  "service_id": "uuid",
  "status_code": 200,
  "response_time_ms": 142,
  "is_success": true,
  "error_message": null,
  "checked_at": "..."
}
```

**Incident**

```json
{
  "id": "uuid",
  "service_id": "uuid",
  "started_at": "...",
  "resolved_at": null,
  "reason": "expected status 200, got 503",
  "is_resolved": false
}
```

---

## Sample API Requests

### Register a service

```bash
curl -s -X POST http://localhost:8080/api/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GitHub API",
    "url": "https://api.github.com",
    "method": "GET",
    "expected_status_code": 200,
    "check_interval_seconds": 30
  }' | jq
```

### List all services

```bash
curl -s http://localhost:8080/api/services | jq
```

### Get one service

```bash
curl -s http://localhost:8080/api/services/<id> | jq
```

### Update a service

```bash
curl -s -X PUT http://localhost:8080/api/services/<id> \
  -H "Content-Type: application/json" \
  -d '{"check_interval_seconds": 60}' | jq
```

### Delete a service

```bash
curl -s -X DELETE http://localhost:8080/api/services/<id>
```

### View recent check results

```bash
curl -s "http://localhost:8080/api/services/<id>/checks?limit=10" | jq
```

### View incidents

```bash
curl -s http://localhost:8080/api/services/<id>/incidents | jq
```

### View summary metrics

```bash
curl -s http://localhost:8080/api/services/<id>/summary | jq
```

---

## Local Setup (Docker Compose)

**Prerequisites:** Docker and Docker Compose installed.

```bash
# 1. Clone the repository
git clone https://github.com/sharonowusu/servicepulse.git
cd servicepulse

# 2. Copy the example environment file
cp .env.example .env

# 3. Start PostgreSQL + the API server
docker compose up --build

# The API is now reachable at http://localhost:8080
```

To stop:

```bash
docker compose down
```

To wipe the database volume:

```bash
docker compose down -v
```

---

## Local Setup (Without Docker)

**Prerequisites:** Go 1.23+, PostgreSQL running locally.

```bash
# 1. Copy and edit the environment file
cp .env.example .env
# Edit DATABASE_URL to point to your local PostgreSQL

# 2. Run the server
go run ./cmd/api
```

---

## Running Tests

```bash
go test ./...
```

To see verbose output and the race detector:

```bash
go test -v -race ./...
```

---

## CI

GitHub Actions runs on every push and pull request to `main`:

- Downloads dependencies
- Runs `go vet`
- Starts a PostgreSQL service container
- Runs all tests with `-race` and generates a coverage report

See [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

---

## Project Structure

```
servicepulse/
  cmd/api/main.go           # main entrypoint
  internal/
    config/config.go        # env var loading
    database/db.go          # connection pool
    database/migrate.go     # SQL migration runner
    models/models.go        # structs + DTOs
    repository/
      interfaces.go         # repository interfaces (enable testability)
      service_repo.go
      check_repo.go
      incident_repo.go
    handlers/
      router.go             # chi router + middleware
      service_handler.go
      check_handler.go
      incident_handler.go
      metrics_handler.go
      response.go           # JSON helpers
    worker/checker.go       # background health checker
  migrations/
    001_create_services.sql
    002_create_check_results.sql
    003_create_incidents.sql
  Dockerfile
  docker-compose.yml
  .env.example
  .github/workflows/ci.yml
```

---

## Future Improvements

- **Authentication** — JWT-based auth so users can manage only their own services
- **Notifications** — Email or webhook alerts when an incident opens or resolves
- **Dashboard frontend** — React or HTMX UI showing real-time service status
- **Prometheus metrics** — Expose a `/metrics` endpoint for Grafana dashboards
- **Retries before incident** — Only open an incident after N consecutive failures
- **Custom request headers** — Allow users to set `Authorization` headers per service
- **Pagination** — Cursor-based pagination for `/checks` and `/incidents`
- **Rate limiting** — Middleware to protect against API abuse
