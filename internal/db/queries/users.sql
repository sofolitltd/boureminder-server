-- name: GetUserByFirebaseUID :one
SELECT * FROM users WHERE firebase_uid = $1;

-- name: CreateUser :one
INSERT INTO users (id, firebase_uid)
VALUES ($1, $2)
ON CONFLICT (firebase_uid) DO UPDATE SET updated_at = now()
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetPartner :one
SELECT u.* FROM users u
JOIN users me ON me.couple_id = u.couple_id
WHERE me.id = $1 AND u.id != $1;

-- name: UpdateProfile :one
UPDATE users
SET display_name   = coalesce(sqlc.narg('display_name'), display_name),
    nickname       = coalesce(sqlc.narg('nickname'), nickname),
    role           = coalesce(sqlc.narg('role'), role),
    birthday_month = coalesce(sqlc.narg('birthday_month'), birthday_month),
    birthday_day   = coalesce(sqlc.narg('birthday_day'), birthday_day),
    birthday_year  = coalesce(sqlc.narg('birthday_year'), birthday_year),
    photo_key      = coalesce(sqlc.narg('photo_key'), photo_key),
    updated_at     = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpsertDevice :exec
INSERT INTO devices (token, user_id, platform, app_version, locale)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (token) DO UPDATE SET
    user_id = $2, platform = $3, app_version = $4, locale = $5, last_seen_at = now();

-- name: DeleteDevice :exec
DELETE FROM devices WHERE token = $1;
