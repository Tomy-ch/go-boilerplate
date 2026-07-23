# Database Initialization SQL

English | [日本語](README.ja.md)

`docker/database/sql/` stores SQL files for **initializing the database environment**.

These files are executed on PostgreSQL container startup via `docker-entrypoint-initdb.d` and perform setup required **before** migrations (database creation, extension installation, etc.).

## Current Files

|File|Description|
|---|---|
|`001-create-local-db.sql`|Create the local development database|
|`002-create-test-db.sql`|Create the test database|
|`003-init-extensions-local-db.sql`|Enable extensions for the local database|
|`004-init-extensions-test-db.sql`|Enable extensions for the test database|

## Execution Order

Files are executed in **ascending order of their numeric prefix** (e.g., `001-...`, `002-...`).

Execution is handled automatically by PostgreSQL's `docker-entrypoint-initdb.d` mechanism on first container startup.

These scripts create/extension-init only the fixed `local` / `test` databases. The DB worktree
pool (`scripts/db-pool`) creates its per-worktree databases (`wt<N>_local` / `wt<N>_test`)
dynamically **after** startup, so it bootstraps the same extensions itself — keep the extension
set here in sync with `scripts/db-pool/pool.sh`. See `docs/maintenance/db-worktree-pool.md`.

## What Belongs Here

- Database creation (`CREATE DATABASE`)
- PostgreSQL extension setup (`CREATE EXTENSION`)
- Roles and permissions required for local/CI environments

## What Does NOT Belong Here

- **DDL (table definitions, schema changes)** → `database/migrations/`
- **DML (data manipulation, queries)** → `database/dml/`
- **Seed data (test/dev initial data)** → `database/seed/`

## Rules

- Use 3-digit numeric prefixes for ordering (e.g., `001-...`, `002-...`)
- Make scripts **idempotent** where possible (`IF NOT EXISTS`) — scripts run on first startup but idempotency helps troubleshooting
- Do not apply directly to CI/production without following team policies
- Keep files to the **minimum required for environment initialization**
