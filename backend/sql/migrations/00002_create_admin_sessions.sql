-- +goose Up
-- token_hash stores SHA-256 of the random session token; the raw token only
-- ever lives in the browser cookie.
CREATE TABLE admin_sessions (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      bigint      NOT NULL REFERENCES admin_users (id) ON DELETE CASCADE,
    token_hash   bytea       NOT NULL,
    csrf_token   text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    ip           inet,
    user_agent   text,
    CONSTRAINT admin_sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT admin_sessions_csrf_not_empty CHECK (length(csrf_token) > 0),
    CONSTRAINT admin_sessions_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT admin_sessions_user_agent_length CHECK (user_agent IS NULL OR length(user_agent) <= 512)
);

CREATE UNIQUE INDEX admin_sessions_token_hash_key ON admin_sessions (token_hash);
CREATE INDEX admin_sessions_user_id_idx ON admin_sessions (user_id);
CREATE INDEX admin_sessions_expires_at_idx ON admin_sessions (expires_at);

-- +goose Down
DROP TABLE admin_sessions;
