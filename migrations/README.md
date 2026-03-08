# Database Migrations

This directory contains SQL migration files for the **file-management-service**.
Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate).

---

## File Naming Convention

golang-migrate requires each migration to have **two files**: an _up_ file and a _down_ file.

```
{version}_{description}.up.sql    # applied when migrating up
{version}_{description}.down.sql  # applied when rolling back
```

`{version}` must be a **monotonically increasing integer** — zero-padded to six digits is recommended:

```
000001_init.up.sql
000001_init.down.sql
000002_seed.up.sql
000002_seed.down.sql
```

The files in this directory (`001_init.sql`, `002_seed.sql`) are the canonical SQL source.
Rename or symlink them to the `NNN_name.up.sql` pattern when registering them with golang-migrate.

---

## Prerequisites

### Install golang-migrate CLI

```bash
# macOS
brew install golang-migrate

# Linux (replace VERSION and ARCH as needed)
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz \
  | tar xvz
sudo mv migrate /usr/local/bin/

# Go install (works everywhere Go is available)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Verify the installation:

```bash
migrate -version
```

---

## Environment Setup

All commands below rely on the `DB_URL` variable exported by the Makefile
(which reads `.env` automatically).  To run commands outside of `make`,
export the variable manually:

```bash
export DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"
```

---

## Running Migrations

### Apply all pending migrations

```bash
make migrate-up
# or directly:
migrate -path migrations -database "${DB_URL}" up
```

### Apply a specific number of steps

```bash
migrate -path migrations -database "${DB_URL}" up 1   # apply only the next migration
```

### Check current migration version

```bash
migrate -path migrations -database "${DB_URL}" version
```

### Force a specific version (use with caution — skips safety checks)

```bash
migrate -path migrations -database "${DB_URL}" force 1
```

---

## Rolling Back Migrations

### Roll back the last applied migration

```bash
make migrate-down
# or directly:
migrate -path migrations -database "${DB_URL}" down 1
```

### Roll back all migrations

```bash
migrate -path migrations -database "${DB_URL}" down
```

> **Warning:** Rolling back `002_seed` removes all seed users, folders, and
> permissions.  Rolling back `001_init` drops every table.  **Never run
> `down` against a production database without a verified backup.**

---

## Creating a New Migration

```bash
migrate create -ext sql -dir migrations -seq {description}
# Example:
migrate create -ext sql -dir migrations -seq add_file_preview_column
```

This generates two empty files:

```
migrations/000003_add_file_preview_column.up.sql
migrations/000003_add_file_preview_column.down.sql
```

Fill in the `up` file with forward DDL/DML changes and the `down` file with
the exact inverse so rollbacks are safe and predictable.

### Template for a new up migration

```sql
-- Migration: 000003_add_file_preview_column
-- Description: <what this migration does and why>

BEGIN;

ALTER TABLE files ADD COLUMN preview_key TEXT;

CREATE INDEX idx_files_preview_key ON files (preview_key)
    WHERE preview_key IS NOT NULL;

COMMIT;
```

### Template for a new down migration

```sql
-- Rollback: 000003_add_file_preview_column

BEGIN;

DROP INDEX IF EXISTS idx_files_preview_key;
ALTER TABLE files DROP COLUMN IF EXISTS preview_key;

COMMIT;
```

---

## Migration Best Practices

| Rule | Rationale |
|------|-----------|
| Wrap DDL in a `BEGIN`/`COMMIT` transaction | Guarantees atomicity; PostgreSQL supports transactional DDL |
| Never edit a migration that has already been applied | Creates schema drift; add a new migration instead |
| Make `down` migrations the exact inverse of `up` | Enables safe, repeatable rollbacks in CI and staging |
| Prefer `ADD COLUMN … DEFAULT NULL` over non-null columns without defaults | Avoids full-table rewrites and lock contention on large tables |
| Use `CONCURRENTLY` for new indexes in production | Prevents read/write blocks: `CREATE INDEX CONCURRENTLY …` |
| Test migrations against a production-sized dataset in staging | Size-dependent issues (lock timeouts, regressions) only surface at scale |
| Back up the database before every production migration | `pg_dump -Fc -f backup_$(date +%Y%m%d_%H%M%S).dump ${DB_NAME}` |

---

## Quick Reference

```bash
# Apply all pending
make migrate-up

# Roll back last step
make migrate-down

# Check current version
migrate -path migrations -database "${DB_URL}" version

# Create next migration
migrate create -ext sql -dir migrations -seq <description>

# Direct psql apply (no migration tracking — dev only)
psql "${DB_URL}" -f migrations/001_init.sql
psql "${DB_URL}" -f migrations/002_seed.sql
```
