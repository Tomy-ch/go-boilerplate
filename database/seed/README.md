# seed

English | [日本語](README.ja.md)

`database/seed` stores **transactional data (seed data) for non-production environments** (local / CI / staging).

## Purpose

Used to insert **initial data** needed to verify application behavior in development, test, and demo environments.

- User data
- Product data
- Other business data

Targets **transactional data**, not master data.

## Execution

```bash
make db-seed
```

Or via CLI:

```bash
./server db-seed
```

## File Naming Convention

```text
{6-digit sequence}_{description}.sql
```

Example:

```text
000001_users.sql
000002_products_electronic_equipment.sql
```

- Executed in ascending order of sequence number
- Control order via sequence numbers when dependencies exist (e.g., users → products)

## Difference from Migrations

|Aspect|migrations|seed|
|---|---|---|
|Target|DDL (schema definitions)|DML (data insertion)|
|Production use|Required|Not recommended|
|Rollback|up / down pairs|None (delete or re-insert data)|
|Idempotency|Required|Recommended (`ON CONFLICT`, etc.)|

## Notes

- **Not intended for production execution** — seed data is for development and testing
- Master data (prefectures, status definitions, etc.) should be managed via migrations
- When tables have foreign key constraints, pay attention to insertion order (sequence numbers)
- For large datasets, split into multiple files and manage with sequence numbers
- Make idempotent where possible (`INSERT ... ON CONFLICT DO NOTHING`, etc.)
