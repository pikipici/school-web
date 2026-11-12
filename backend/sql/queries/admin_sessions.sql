-- name: CreateAdminSession :one
INSERT INTO admin_sessions (user_id, token_hash, csrf_token, created_at, last_seen_at, expires_at, ip, user_agent)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(token_hash),
    sqlc.arg(csrf_token),
    sqlc.arg(created_at),
    sqlc.arg(created_at),
    sqlc.arg(expires_at),
    NULLIF(sqlc.arg(ip)::text, '')::inet,
    NULLIF(sqlc.arg(user_agent)::text, '')
)
RETURNING id;

-- name: GetAdminSessionByTokenHash :one
SELECT s.id, s.user_id, u.username, s.csrf_token, s.created_at, s.last_seen_at, s.expires_at
FROM admin_sessions s
JOIN admin_users u ON u.id = s.user_id
WHERE s.token_hash = $1;

-- name: TouchAdminSession :exec
UPDATE admin_sessions SET last_seen_at = $2 WHERE id = $1;

-- name: DeleteAdminSessionByID :exec
DELETE FROM admin_sessions WHERE id = $1;

-- name: DeleteAdminSessionByTokenHash :exec
DELETE FROM admin_sessions WHERE token_hash = $1;

-- name: DeleteAdminSessionsByUser :exec
DELETE FROM admin_sessions WHERE user_id = $1;

-- name: DeleteStaleAdminSessions :execrows
DELETE FROM admin_sessions
WHERE expires_at <= sqlc.arg(now)::timestamptz
   OR last_seen_at <= sqlc.arg(idle_cutoff)::timestamptz;
