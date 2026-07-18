-- +goose Up
CREATE TABLE games(
    id UUID PRIMARY KEY,
    map_index INTEGER NOT NULL,
    winner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    duration_seconds INTEGER NOT NULL,
    played_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE game_users (
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    hidden_seconds INTEGER NOT NULL,
    ranking INTEGER NOT NULL,

    PRIMARY KEY (game_id, user_id)
);

-- +goose Down
DROP TABLE game_users;
DROP TABLE games;
