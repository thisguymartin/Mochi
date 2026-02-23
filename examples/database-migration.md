# Database Migration: Users Table to PostgreSQL

Migrate the users table from SQLite to PostgreSQL with zero downtime.

## Current State

- SQLite database at `data/app.db`
- `users` table with columns: `id INTEGER PRIMARY KEY`, `email TEXT UNIQUE`, `name TEXT`, `password_hash TEXT`, `created_at TEXT`, `updated_at TEXT`
- Approximately 50,000 rows in production

## Target State

- PostgreSQL 16 database
- Proper column types: `id SERIAL PRIMARY KEY`, `email VARCHAR(255) UNIQUE`, `name VARCHAR(100)`, `password_hash VARCHAR(60)`, `created_at TIMESTAMPTZ DEFAULT NOW()`, `updated_at TIMESTAMPTZ DEFAULT NOW()`
- Add indices on `email` and `created_at`

## Migration Steps

1. **Create migration file** using the project's migration tool
2. **Write the up migration** - create the new table with proper types and indices
3. **Write the down migration** - drop the table
4. **Create a data migration script** that reads from SQLite and inserts into PostgreSQL
   - Handle timestamp format conversion (SQLite TEXT -> PostgreSQL TIMESTAMPTZ)
   - Batch inserts (500 rows per batch) for performance
   - Validate row counts match after migration
5. **Update the database connection config** to support both SQLite and PostgreSQL
6. **Update all repository/DAO code** to use PostgreSQL-compatible queries

## Constraints

- Do NOT use an ORM - use raw SQL with parameterized queries
- Preserve all existing user IDs
- The migration script should be idempotent (safe to re-run)
- Add a rollback mechanism
