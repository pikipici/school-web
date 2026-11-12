-- name: GetAdminUserByUsername :one
SELECT id, username, password_hash
FROM admin_users
WHERE username = $1;

-- name: TouchAdminUserLogin :exec
UPDATE admin_users SET last_login_at = $2, updated_at = $2 WHERE id = $1;

-- name: UpdateAdminUserPasswordHash :exec
UPDATE admin_users SET password_hash = $2, updated_at = $3 WHERE id = $1;

-- name: CreateFirstAdminUser :one
-- Inserts only while the table is empty, so the site can never have more than one admin.
INSERT INTO admin_users (username, password_hash)
SELECT sqlc.arg(username)::text, sqlc.arg(password_hash)::text
WHERE NOT EXISTS (SELECT 1 FROM admin_users)
RETURNING id;
