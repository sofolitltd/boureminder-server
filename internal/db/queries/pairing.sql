-- name: CreateCouple :one
INSERT INTO couples (id) VALUES ($1) RETURNING *;

-- name: GetCouple :one
SELECT * FROM couples WHERE id = $1;

-- name: SetWeddingDate :one
UPDATE couples
SET wedding_month = $2, wedding_day = $3, wedding_year = $4,
    wedding_set_by = $5, wedding_confirmed = $6, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ConfirmWeddingDate :one
UPDATE couples SET wedding_confirmed = true, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: LockPairCode :one
SELECT owner_id, expires_at, used_at FROM pair_codes WHERE code = $1 FOR UPDATE;

-- name: CreatePairCode :one
INSERT INTO pair_codes (code, owner_id, expires_at) VALUES ($1, $2, $3) RETURNING *;

-- name: ExpireUnusedPairCodesForOwner :exec
UPDATE pair_codes SET expires_at = now() WHERE owner_id = $1 AND used_at IS NULL;

-- name: MarkPairCodeUsed :exec
UPDATE pair_codes SET used_at = now() WHERE code = $1;

-- name: CountPairedAmong :one
-- FOR UPDATE can't combine directly with an aggregate, so the lock lives
-- in the subquery and the count wraps it.
SELECT count(*) FROM (
    SELECT id, couple_id FROM users WHERE id = ANY($1::uuid[]) FOR UPDATE
) locked WHERE couple_id IS NOT NULL;

-- name: AssignCoupleToUsers :exec
UPDATE users SET couple_id = sqlc.arg('couple_id'), updated_at = now()
WHERE id = ANY(sqlc.arg('user_ids')::uuid[]);
