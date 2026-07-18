-- name: CreateUser :one
INSERT INTO users(id, username, password_hash)
VALUES ($1,
        $2,
        $3)
RETURNING *;