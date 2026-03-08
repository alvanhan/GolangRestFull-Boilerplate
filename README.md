# File Management Service

An enterprise-grade backend service for secure document management with strict role-based access control, chunked file uploads, real-time notifications, and background job processing. Designed for organizations requiring high-security storage of critical documents.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Technology Stack](#technology-stack)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Environment Configuration](#environment-configuration)
- [Installation](#installation)
- [Running the Service](#running-the-service)
- [API Documentation](#api-documentation)
- [Database](#database)
- [File Upload](#file-upload)
- [Real-time Notifications](#real-time-notifications)
- [Background Jobs](#background-jobs)
- [Role-Based Access Control](#role-based-access-control)
- [Development](#development)
- [Makefile Reference](#makefile-reference)
- [Service Credentials](#service-credentials)
- [Security Considerations](#security-considerations)

---

## Architecture Overview

This service follows **Clean Architecture** principles, separating concerns into four independent layers:

```
┌─────────────────────────────────────────────────────────────┐
│                      Delivery Layer                         │
│         HTTP Handlers / Middleware / Router (Fiber)         │
├─────────────────────────────────────────────────────────────┤
│                     Use Case Layer                          │
│      Business Logic: Auth / File / Folder / Permission      │
├─────────────────────────────────────────────────────────────┤
│                      Domain Layer                           │
│         Entities / Repository Interfaces / Errors           │
├─────────────────────────────────────────────────────────────┤
│                  Infrastructure Layer                       │
│    PostgreSQL / Redis / MinIO / Worker / Scheduler / Cache  │
└─────────────────────────────────────────────────────────────┘
```

Dependencies point inward. The domain layer has no external dependencies.

---

## Technology Stack

| Component | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP Framework | Fiber v2 |
| Database | PostgreSQL 16 |
| Cache / Pub-Sub | Redis 7 |
| Object Storage | MinIO (S3-compatible) |
| ORM | GORM |
| Authentication | JWT (access + refresh token) |
| Background Jobs | Asynq (Redis-backed) |
| Cron Scheduler | robfig/cron v3 |
| Configuration | Viper |
| Logging | Zap (structured JSON) |
| Validation | go-playground/validator v10 |
| API Documentation | Swagger UI (swaggo) |
| Containerization | Docker / Docker Compose |

---

## Project Structure

```
.
├── cmd/
│   └── api/
│       ├── main.go              # Entry point, dependency injection
│       └── adapters.go          # Interface adapters (storage, worker, notification)
├── config/
│   └── config.go                # Viper configuration singleton
├── docker/
│   └── Dockerfile               # Multi-stage build (golang:1.25-alpine -> alpine:3.19)
├── docs/                        # Auto-generated Swagger documentation
├── internal/
│   ├── delivery/
│   │   └── http/
│   │       ├── handler/         # HTTP request handlers (auth, file, folder, permission, etc.)
│   │       ├── middleware/      # Auth, RBAC, rate limiter, request logger
│   │       └── router/          # Fiber app setup and route registration
│   ├── domain/
│   │   ├── entity/              # Core entities: User, File, Folder, Permission, AuditLog
│   │   ├── errors/              # Sentinel errors and AppError type
│   │   └── repository/          # Repository interfaces
│   ├── infrastructure/
│   │   ├── cache/               # Redis cache implementation
│   │   ├── database/            # PostgreSQL and Redis connection setup
│   │   ├── notification/        # Redis Pub/Sub publisher
│   │   ├── repository/          # GORM repository implementations
│   │   ├── storage/             # MinIO storage implementation
│   │   └── worker/              # Asynq task definitions, client, processor, scheduler
│   └── usecase/
│       ├── auth/                # Authentication use case
│       ├── file/                # File management use case
│       ├── folder/              # Folder management use case
│       ├── permission/          # Permission management use case
│       ├── notification/        # Notification use case
│       ├── audit/               # Audit log use case
│       └── admin/               # Admin management use case
├── migrations/
│   ├── 001_init.sql             # Full schema: 10 tables, 46 indexes, triggers
│   └── 002_seed.sql             # Seed data: default admin user
├── pkg/
│   ├── crypto/                  # Password hashing (bcrypt)
│   ├── jwt/                     # JWT generation and validation
│   ├── logger/                  # Zap logger factory
│   ├── pagination/              # Pagination helpers
│   ├── response/                # Standardized HTTP response helpers
│   ├── utils/                   # General utilities
│   └── validator/               # Request struct validation
├── .env.example                 # Environment variable template
├── docker-compose.yml           # All services: api, postgres, redis, minio
├── Makefile                     # Development task runner
└── go.mod
```

---

## Prerequisites

The following must be installed on the host machine:

| Tool | Minimum Version | Purpose |
|---|---|---|
| Go | 1.25 | Building the service locally |
| Docker | 24.x | Container runtime |
| Docker Compose | v2.x | Multi-container orchestration |
| Make | Any | Task runner (optional) |
| golang-migrate | v4 | Running SQL migrations manually |

Install `golang-migrate`:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Install `swag` (for regenerating API docs):
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## Environment Configuration

Copy the example environment file and fill in the values:

```bash
cp .env.example .env
```

### Critical Variables

The following variables **must** be changed before running in any environment:

| Variable | Description | Example |
|---|---|---|
| `JWT_ACCESS_SECRET` | Secret key for signing access tokens | `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET` | Secret key for signing refresh tokens | `openssl rand -hex 32` |
| `DB_PASSWORD` | PostgreSQL password | Strong password |
| `REDIS_PASSWORD` | Redis password | Strong password |
| `MINIO_ACCESS_KEY` | MinIO root user | Strong username |
| `MINIO_SECRET_KEY` | MinIO root password | Strong password (min 8 chars) |

Generate secure secrets:
```bash
make gen-secret
```

### Full Environment Reference

```env
# Application
APP_NAME=file-management-service
APP_ENV=development          # development | production
APP_PORT=8080
APP_DEBUG=true

# JWT
JWT_ACCESS_SECRET=           # Required: generate with `make gen-secret`
JWT_REFRESH_SECRET=          # Required: generate with `make gen-secret`
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=                 # Required
DB_NAME=file_management
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=25
DB_MAX_LIFETIME=5m

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=              # Required
REDIS_DB=0
REDIS_POOL_SIZE=10

# MinIO (Object Storage)
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=            # Required
MINIO_SECRET_KEY=            # Required
MINIO_BUCKET_NAME=documents
MINIO_USE_SSL=false
MINIO_REGION=us-east-1

# Upload
UPLOAD_MAX_SIZE=104857600    # 100 MB in bytes
UPLOAD_CHUNK_SIZE=10485760   # 10 MB per chunk
UPLOAD_TEMP_DIR=/tmp/uploads
UPLOAD_ALLOWED_TYPES=pdf,doc,docx,xls,xlsx,ppt,pptx,jpg,jpeg,png,gif,txt,zip,rar

# Rate Limiting
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m

# Worker
WORKER_CONCURRENCY=10
WORKER_QUEUE_DEFAULT=default
WORKER_QUEUE_CRITICAL=critical

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Origin,Content-Type,Accept,Authorization,X-Request-ID
```

---

## Installation

### Step 1 — Clone the repository

```bash
git clone https://github.com/alvanhan/GolangRestFull-Boilerplate
cd GolangRestFull-Boilerplate
```

### Step 2 — Configure environment

```bash
cp .env.example .env
```

Edit `.env` and set all required values (see [Critical Variables](#critical-variables)).

### Step 3 — Build Docker image

```bash
docker compose build --no-cache
```

This compiles the Go binary inside a Docker build container and produces a minimal Alpine-based image (~20MB).

### Step 4 — Start all services

```bash
docker compose up -d
```

This starts the following containers:
- `file-management-api` — Go application on port `8080`
- `file-management-postgres` — PostgreSQL 16 on port `5432`
- `file-management-redis` — Redis 7 on port `6379`
- `file-management-minio` — MinIO on ports `9000` (API) and `9001` (Console)

### Step 5 — Verify services

```bash
docker ps
```

All containers should show `(healthy)` status.

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "service": "file-management-service",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## Running the Service

### Start (all services)

```bash
docker compose up -d
```

### Stop (preserve data)

```bash
docker compose down
```

### Stop and remove all data (full reset)

```bash
docker compose down -v --remove-orphans
```

> **Warning:** The `-v` flag deletes all Docker volumes. All database data, Redis data, and stored files will be permanently deleted.

### Rebuild API only (after code changes)

```bash
docker compose build api --no-cache
docker compose up -d api
```

### View logs

```bash
# All services
docker compose logs -f

# API only
make docker-logs
```

---

## API Documentation

Interactive Swagger UI is available at:

```
http://localhost:8080/swagger/index.html
```

To authenticate in Swagger UI:
1. Call `POST /api/v1/auth/login` to obtain an access token.
2. Click the **Authorize** button in the top-right of the Swagger UI.
3. Enter: `Bearer <your-access-token>`

### Endpoint Groups

| Group | Base Path | Description |
|---|---|---|
| Auth | `/api/v1/auth` | Register, login, token refresh, profile |
| Files | `/api/v1/files` | Upload, download, move, copy, delete, share, versioning |
| Folders | `/api/v1/folders` | Create, list, tree, breadcrumb, move, share |
| Permissions | `/api/v1/permissions` | Grant, revoke, bulk grant, check access |
| Notifications | `/api/v1/notifications` | List, SSE stream, mark as read |
| Audit Logs | `/api/v1/audit-logs` | List and export (admin only) |
| Admin | `/api/v1/admin` | User management, system statistics |

Regenerate documentation after modifying handler annotations:

```bash
make swag
```

---

## Database

The service uses **GORM AutoMigrate** on startup for development convenience. For production, use the SQL migration files.

### Run SQL Migrations

```bash
# Apply all migrations
make migrate-up

# Rollback last migration
make migrate-down
```

### Seed Data

`migrations/002_seed.sql` inserts a default admin user:

| Field | Value |
|---|---|
| Email | `admin@filemanagement.com` |
| Password | `Admin@123456` |
| Role | `admin` |

> Change the admin password immediately after first login in any non-development environment.

### Schema Overview

| Table | Description |
|---|---|
| `users` | User accounts with role and status |
| `files` | File metadata, versioning, share links |
| `file_versions` | Version history for each file |
| `file_chunks` | Chunk tracking for multipart uploads |
| `folders` | Folder hierarchy using materialized paths |
| `permissions` | Resource-level access control (polymorphic) |
| `audit_logs` | Full audit trail for all operations |
| `notifications` | Per-user notification records |
| `refresh_tokens` | JWT refresh token store |
| `share_links` | Time-limited public share tokens |

---

## File Upload

### Standard Upload (up to configured max size)

```http
POST /api/v1/files/upload
Content-Type: multipart/form-data
Authorization: Bearer <token>

file=<binary>
folder_id=<uuid>  (optional)
```

### Chunked Upload (for large files)

**Step 1 — Initialize upload session**

```http
POST /api/v1/files/upload/init
Content-Type: application/json

{
  "filename": "document.pdf",
  "total_size": 524288000,
  "total_chunks": 50,
  "folder_id": "<uuid>"
}
```

**Step 2 — Upload each chunk**

```http
POST /api/v1/files/upload/chunk
Content-Type: multipart/form-data

upload_id=<session-id>
chunk_index=0
chunk=<binary>
```

**Step 3 — Complete upload**

```http
POST /api/v1/files/upload/complete
Content-Type: application/json

{
  "upload_id": "<session-id>"
}
```

---

## Real-time Notifications

The service exposes a **Server-Sent Events (SSE)** endpoint for real-time push notifications:

```
GET /api/v1/notifications/stream
Authorization: Bearer <token>
```

The client receives events in the following format:

```
event: notification
data: {"id":"...","type":"file_shared","message":"...","created_at":"..."}

event: heartbeat
data: ping
```

Notifications are delivered via **Redis Pub/Sub**, with a dedicated channel per user: `notifications:{userID}`.

---

## Background Jobs

Background tasks are processed by **Asynq** (Redis-backed queue):

| Task | Description |
|---|---|
| `file:process` | Post-upload processing (virus scan hook, thumbnail) |
| `file:cleanup` | Remove orphaned temp chunks |
| `notification:send` | Async notification dispatch |
| `audit:log` | Async audit log persistence |

### Cron Jobs

Scheduled tasks run via **robfig/cron**:

| Schedule | Task |
|---|---|
| Every hour | Clean up expired share links |
| Every day at 02:00 | Remove soft-deleted files older than 30 days |
| Every day at 03:00 | Generate storage usage statistics |

---

## Role-Based Access Control

The system enforces two levels of access control:

### System Roles

| Role | Capabilities |
|---|---|
| `admin` | Full access to all resources, user management, audit logs, system stats |
| `user` | Access only to owned resources and explicitly shared resources |

### Resource-Level Permissions

Each file and folder can have per-user permissions assigned independently of system role:

| Permission | Description |
|---|---|
| `read` | View file metadata and download content |
| `write` | Upload, edit, rename files |
| `delete` | Delete files and folders |
| `share` | Create share links and grant access to others |
| `admin` | Full control over the resource |

Permissions are inherited from parent folders. Explicit resource permissions override inherited ones.

---

## Development

### Run locally (without Docker)

Ensure PostgreSQL, Redis, and MinIO are running and configured in `.env`, then:

```bash
go run ./cmd/api/
```

Or using Make:

```bash
make run
```

### Build binary

```bash
make build
# Output: bin/api
```

### Run tests

```bash
# All tests with race detector and coverage
make test

# Unit tests only
make test-unit
```

### Generate mocks

```bash
make mock
```

---

## Makefile Reference

| Target | Description |
|---|---|
| `make run` | Run the service with `go run` |
| `make build` | Compile binary to `bin/api` |
| `make test` | Run all tests with coverage report |
| `make test-unit` | Run unit tests only |
| `make migrate-up` | Apply all pending SQL migrations |
| `make migrate-down` | Rollback the last migration |
| `make docker-up` | Start all Docker services |
| `make docker-down` | Stop all Docker services |
| `make docker-logs` | Follow API container logs |
| `make swag` | Regenerate Swagger documentation |
| `make mock` | Generate interface mocks |
| `make lint` | Run golangci-lint |
| `make tidy` | Run `go mod tidy` and verify |
| `make gen-secret` | Generate a random 32-byte hex secret |
| `make clean` | Remove build artifacts |

---

## Service Credentials

Default credentials for local development. All values are configurable via `.env`.

### PostgreSQL

| Parameter | Value |
|---|---|
| Host | `localhost` |
| Port | `5432` |
| Database | `file_management` |
| Username | `postgres` |
| Password | `FileManager@2024` |

### Redis

| Parameter | Value |
|---|---|
| Host | `localhost` |
| Port | `6379` |
| Password | `Redis@2024` |
| Database index | `0` |

### MinIO

| Parameter | Value |
|---|---|
| API Endpoint | `localhost:9000` |
| Console URL | `http://localhost:9001` |
| Access Key | `minioadmin` |
| Secret Key | `minioadmin123` |
| Default Bucket | `documents` |

> These are development defaults. Never use these values in staging or production environments.

---

## Security Considerations

The following hardening steps are mandatory before deploying to any production or staging environment:

1. **Rotate all secrets** — Generate new values for `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `DB_PASSWORD`, `REDIS_PASSWORD`, `MINIO_ACCESS_KEY`, and `MINIO_SECRET_KEY`.

2. **Enable TLS** — Place the service behind a reverse proxy (Nginx, Caddy, or a cloud load balancer) with TLS termination. Set `MINIO_USE_SSL=true` when using an external MinIO deployment.

3. **Restrict CORS** — Set `CORS_ALLOWED_ORIGINS` to the exact frontend domain. Remove the wildcard `*`.

4. **Enable PostgreSQL SSL** — Set `DB_SSLMODE=require` and provision certificates.

5. **Change admin password** — The seed admin password `Admin@123456` must be changed immediately after the first deployment.

6. **Set `APP_ENV=production`** — Disables stack traces in error responses and verbose route printing.

7. **Apply SQL migrations** — Run `make migrate-up` instead of relying on GORM AutoMigrate for schema management.

8. **Restrict network access** — PostgreSQL, Redis, and MinIO ports (`5432`, `6379`, `9000`, `9001`) should not be exposed to the public internet. Use Docker internal networks or firewall rules.

9. **Configure rate limiting** — Tune `RATE_LIMIT_MAX` and `RATE_LIMIT_WINDOW` based on expected traffic patterns.

10. **Review audit logs** — The audit log system records all create, update, delete, and access operations. Set up alerts on suspicious patterns.

---

## License

This project is proprietary and confidential. Unauthorized copying, distribution, or use is strictly prohibited.
