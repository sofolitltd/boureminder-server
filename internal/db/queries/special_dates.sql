-- name: CreateSpecialDate :one
INSERT INTO special_dates (
    id, couple_id, kind, title, emoji, month, day, year,
    repeats_yearly, notify_user_ids, is_system, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: ListSpecialDates :many
SELECT * FROM special_dates WHERE couple_id = $1 AND deleted_at IS NULL ORDER BY month, day;
