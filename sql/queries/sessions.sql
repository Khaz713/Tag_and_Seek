-- name: CreateSession :one
INSERT INTO sessions(user_id)
VALUES ($1 )
RETURNING *;

-- name: DeleteSessionByToken :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();