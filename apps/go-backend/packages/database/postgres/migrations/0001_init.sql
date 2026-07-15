-- +migrate Up
-- This file is identical to packages/database/postgres/schema.sql and is kept
-- so a migration tool (e.g. golang-migrate) can version schema changes going
-- forward. schema.sql remains the fast-path used by docker-entrypoint-initdb.d
-- for local development; new changes should be added as 0002_*.sql, 0003_*.sql, etc.

\i ../schema.sql
