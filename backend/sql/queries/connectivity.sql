-- Infrastructure-only sqlc contract; this is not business schema or a domain query.
-- name: ConnectivityCheck :one
SELECT 1::integer AS ok;
