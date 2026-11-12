-- name: GetAdminUserByUsername :one
SELECT id, username, password_hash
FROM admin_users
WHERE username = $1;

-- name: TouchAdminUserLogin :exec
UPDATE admin_users SET last_login_at = $2, updated_at = $2 WHERE id = $1;

-- name: UpdateAdminUserPasswordHash :exec
UPDATE admin_users SET password_hash = $2, updated_at = $3 WHERE id = $1;
