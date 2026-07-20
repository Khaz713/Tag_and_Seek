-- name: CreateGame :one
INSERT INTO games(id, map_index, winner_id, duration_seconds)
VALUES ($1,
        $2,
        $3,
        $4)
RETURNING *;

-- name: CreateGameUser :one
INSERT INTO game_users(game_id, user_id, hidden_seconds, ranking)
VALUES (
        $1,
        $2,
        $3,
        $4
       )
RETURNING *;

-- name: GetGamesByUserID :many
SELECT *
FROM games g
JOIN game_users gu ON g.id = gu.game_id
WHERE gu.user_id = $1
ORDER BY g.played_at DESC;

-- name: GetParticipantsByGameID :many
SELECT
    u.id AS user_id,
    u.username,
    gu.hidden_seconds,
    gu.ranking
FROM game_users gu
         JOIN users u ON gu.user_id = u.id
WHERE gu.game_id = $1
ORDER BY gu.ranking ASC;
