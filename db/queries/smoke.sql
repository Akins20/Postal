-- Trivial query proving the sqlc generation chain works. Replaced by real
-- domain queries from Phase 2 onward.

-- name: InsertSmoke :one
INSERT INTO schema_smoke (note)
VALUES ($1)
RETURNING id, note, created_at;

-- name: CountSmoke :one
SELECT count(*) FROM schema_smoke;
