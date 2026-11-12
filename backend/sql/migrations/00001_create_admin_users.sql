-- +goose Up
CREATE TABLE admin_users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      text        NOT NULL,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz,
    CONSTRAINT admin_users_username_format CHECK (username ~ '^[a-z0-9._-]{3,32}$'),
    CONSTRAINT admin_users_password_hash_not_empty CHECK (length(password_hash) > 0)
);

CREATE UNIQUE INDEX admin_users_username_key ON admin_users (username);

-- +goose Down
DROP TABLE admin_users;
