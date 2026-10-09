-- name: CreateDeedVault :one
INSERT INTO deed_vaults (author_id, couple_id, recipient_id, unlock_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDeedVault :one
SELECT * FROM deed_vaults WHERE author_id = $1;
