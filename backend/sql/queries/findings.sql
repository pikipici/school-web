-- name: ListFindings :many
SELECT * FROM evaluation_findings WHERE group_id = $1 ORDER BY id;

-- name: GetFinding :one
SELECT * FROM evaluation_findings WHERE id = $1;

-- name: InsertFinding :one
INSERT INTO evaluation_findings (group_id, file_type, location, problem, required_action, priority, created_by, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING *;

-- name: UpdateFinding :one
UPDATE evaluation_findings SET file_type=$3, location=$4, problem=$5, required_action=$6, priority=$7, resolved=$8, updated_at=$9
WHERE id=$1 AND updated_at=$2
AND (published_at IS NULL OR (file_type=$3 AND location=$4 AND problem=$5 AND required_action=$6 AND priority=$7)) RETURNING *;

-- name: DeleteDraftFinding :execrows
DELETE FROM evaluation_findings WHERE id=$1 AND updated_at=$2 AND published_at IS NULL;

-- name: PublishFindings :execrows
UPDATE evaluation_findings SET published_at=$2, updated_at=$2 WHERE group_id=$1 AND published_at IS NULL;

-- name: GetReview :one
SELECT review_status, review_updated_at FROM study_groups WHERE id=$1;

-- name: SetReview :one
UPDATE study_groups SET review_status=$3, review_updated_at=$4 WHERE id=$1 AND review_updated_at=$2
RETURNING review_status, review_updated_at;

-- name: ReviewStateValid :one
SELECT CASE review_status
WHEN 'DISETUJUI' THEN
    coalesce((SELECT bool_or(ppt_received) AND bool_or(pdf_received) FROM submission_rounds WHERE group_id=g.id),false)
    AND NOT EXISTS (SELECT 1 FROM evaluation_findings WHERE group_id=g.id AND priority='WAJIB' AND NOT resolved)
WHEN 'PERLU_REVISI' THEN
    EXISTS (SELECT 1 FROM evaluation_findings WHERE group_id=g.id AND priority='WAJIB' AND NOT resolved AND published_at IS NOT NULL)
ELSE true END::boolean AS valid
FROM study_groups g WHERE g.id=$1;

-- name: ListPublicFindings :many
SELECT id, file_type, location, problem, required_action, priority, resolved, published_at, updated_at
FROM evaluation_findings WHERE group_id=$1 AND published_at IS NOT NULL ORDER BY id;

-- name: GetPublicGroup :one
SELECT g.id, g.name, g.review_status,
       greatest(g.updated_at, coalesce(g.progress_updated_at,g.updated_at))::timestamptz AS updated_at,
       c.code AS class_code, t.code AS topic_code, t.title AS topic_title,
       coalesce((SELECT bool_or(r.ppt_received) FROM submission_rounds r WHERE r.group_id=g.id),false)::boolean AS has_ppt,
       coalesce((SELECT bool_or(r.pdf_received) FROM submission_rounds r WHERE r.group_id=g.id),false)::boolean AS has_pdf,
       coalesce((SELECT n.is_received FROM group_notebooks n WHERE n.group_id=g.id),false)::boolean AS notebook_received
FROM study_groups g JOIN classes c ON c.id=g.class_id JOIN academic_years y ON y.id=c.academic_year_id JOIN topics t ON t.id=g.topic_id
WHERE g.id=$1 AND g.is_active AND c.is_active AND y.is_active;
