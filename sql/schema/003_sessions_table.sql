-- +goose Up
CREATE TABLE sessions
(
    token UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

-- +goose Down
DROP TABLE sessions;