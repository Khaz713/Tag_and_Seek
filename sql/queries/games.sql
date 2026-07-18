-- name: CreateGame :one
INSERT INTO games(id, map_index, winner_id, duration_seconds)
VALUES ($1,
        $2,
        $3,
        $4)
RETURNING *;

-- name: CreateGamePlayer :one
INSERT INTO game_users(game_id, user_id, hidden_seconds, ranking)
VALUES (
        $1,
        $2,
        $3,
        $4
       )
RETURNING *;