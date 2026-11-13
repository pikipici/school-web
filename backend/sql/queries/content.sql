-- name: CreateAnnouncement :one
INSERT INTO announcements (title, body, status, pinned, published_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, CASE WHEN $3 = 'published' THEN $5::timestamptz ELSE NULL::timestamptz END, $5, $5)
RETURNING *;

-- name: GetAnnouncement :one
SELECT * FROM announcements WHERE id = $1;

-- name: UpdateAnnouncement :one
UPDATE announcements
SET title=$3, body=$4, status=$5, pinned=$6,
    published_at=CASE WHEN $5 = 'published' THEN COALESCE(published_at, $7::timestamptz) ELSE published_at END,
    updated_at=$7
WHERE id=$1 AND updated_at=$2
RETURNING *;

-- name: CreateMaterial :one
INSERT INTO materials (topic_id, title, description, status, sort_order, published_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, CASE WHEN $4 = 'published' THEN $6::timestamptz ELSE NULL::timestamptz END, $6, $6)
RETURNING *;

-- name: GetMaterial :one
SELECT * FROM materials WHERE id = $1;

-- name: UpdateMaterial :one
UPDATE materials
SET topic_id=$3, title=$4, description=$5, status=$6, sort_order=$7,
    published_at=CASE WHEN $6 = 'published' THEN COALESCE(published_at, $8::timestamptz) ELSE published_at END,
    updated_at=$8
WHERE id=$1 AND updated_at=$2
RETURNING *;

-- name: TopicExists :one
SELECT EXISTS(SELECT 1 FROM topics WHERE id=$1);
