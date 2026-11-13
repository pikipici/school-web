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

-- name: ListPublicAnnouncements :many
SELECT id, title, body, published_at FROM announcements
WHERE status = 'published'
ORDER BY pinned DESC, published_at DESC, id DESC LIMIT $1 OFFSET $2;

-- name: CountPublicAnnouncements :one
SELECT count(*) FROM announcements WHERE status = 'published';

-- name: GetPublicAnnouncement :one
SELECT id, title, body, published_at FROM announcements WHERE id=$1 AND status='published';

-- name: ListPublicTopics :many
SELECT id, code, title, sort_order FROM topics ORDER BY sort_order, id;

-- name: ListPublicMaterials :many
SELECT m.id, m.title, m.description, m.published_at, t.id AS topic_id, t.code AS topic_code, t.title AS topic_title
FROM materials m LEFT JOIN topics t ON t.id=m.topic_id
WHERE m.status='published' AND (sqlc.narg(topic_code)::text IS NULL OR t.code=sqlc.narg(topic_code))
ORDER BY t.sort_order NULLS LAST, m.sort_order, m.id LIMIT $1 OFFSET $2;

-- name: CountPublicMaterials :one
SELECT count(*) FROM materials m LEFT JOIN topics t ON t.id=m.topic_id
WHERE m.status='published' AND (sqlc.narg(topic_code)::text IS NULL OR t.code=sqlc.narg(topic_code));

-- name: GetPublicMaterial :one
SELECT m.id, m.title, m.description, m.published_at, t.id AS topic_id, t.code AS topic_code, t.title AS topic_title
FROM materials m LEFT JOIN topics t ON t.id=m.topic_id WHERE m.id=$1 AND m.status='published';
