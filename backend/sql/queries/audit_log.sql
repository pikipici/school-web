-- name: InsertAuditLog :exec
INSERT INTO audit_log (occurred_at, actor_user_id, action, outcome, ip, detail)
VALUES (
    sqlc.arg(occurred_at),
    sqlc.narg(actor_user_id),
    sqlc.arg(action),
    sqlc.arg(outcome),
    NULLIF(sqlc.arg(ip)::text, '')::inet,
    sqlc.arg(detail)::jsonb
);
