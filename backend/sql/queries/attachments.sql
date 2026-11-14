-- name: MaterialExists :one
SELECT EXISTS(SELECT 1 FROM materials WHERE id=$1);

-- name: AttachmentUsage :one
SELECT
    (SELECT coalesce(sum(a.size_bytes), 0) FROM material_attachments a)::bigint AS total_bytes,
    (SELECT count(*) FROM material_attachments m WHERE m.material_id = sqlc.arg(material_id)::bigint)::bigint AS material_count;

-- name: InsertAttachment :one
INSERT INTO material_attachments (material_id, storage_key, original_name, kind, content_type, size_bytes, sha256, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListMaterialAttachments :many
SELECT * FROM material_attachments WHERE material_id=$1 ORDER BY id;

-- name: DeleteAttachment :one
DELETE FROM material_attachments WHERE id=$1 RETURNING *;

-- name: LockMaterialTitle :one
SELECT title FROM materials WHERE id=$1 FOR UPDATE;

-- name: MaterialAttachmentKeys :many
SELECT storage_key FROM material_attachments WHERE material_id=$1;
