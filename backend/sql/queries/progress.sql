-- name: ListAcademicYears :many
SELECT * FROM academic_years ORDER BY label DESC;

-- name: CreateAcademicYear :one
INSERT INTO academic_years (label, created_at, updated_at) VALUES ($1, $2, $2) RETURNING *;

-- name: AcademicYearExists :one
SELECT EXISTS(SELECT 1 FROM academic_years WHERE id = $1);

-- name: DeactivateAcademicYears :exec
UPDATE academic_years SET is_active = false, updated_at = $1 WHERE is_active;

-- name: ActivateAcademicYear :execrows
UPDATE academic_years SET is_active = true, updated_at = $2 WHERE id = $1;

-- name: ListClasses :many
SELECT c.*, (SELECT count(*) FROM study_groups g WHERE g.class_id = c.id)::bigint AS group_count
FROM classes c WHERE c.academic_year_id = $1
ORDER BY lower(c.code), c.id;

-- name: GetClass :one
SELECT c.*, (SELECT count(*) FROM study_groups g WHERE g.class_id = c.id)::bigint AS group_count
FROM classes c WHERE c.id = $1;

-- name: ClassExists :one
SELECT EXISTS(SELECT 1 FROM classes WHERE id = $1);

-- name: CreateClass :one
INSERT INTO classes (academic_year_id, code, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $4) RETURNING *;

-- name: UpdateClass :one
UPDATE classes SET academic_year_id = $3, code = $4, name = $5, is_active = $6, updated_at = $7
WHERE id = $1 AND updated_at = $2 RETURNING *;

-- name: DeleteClass :one
DELETE FROM classes WHERE id = $1 RETURNING code;

-- name: ListGroups :many
SELECT * FROM study_groups WHERE class_id = $1 ORDER BY sort_order, lower(name), id;

-- name: GetGroup :one
SELECT * FROM study_groups WHERE id = $1;

-- name: CreateGroup :one
INSERT INTO study_groups (class_id, topic_id, name, expected_email, sort_order, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6) RETURNING *;

-- name: UpdateGroup :one
UPDATE study_groups
SET class_id = $3, topic_id = $4, name = $5, expected_email = $6, sort_order = $7, is_active = $8, updated_at = $9
WHERE id = $1 AND updated_at = $2 RETURNING *;

-- name: DeleteGroup :one
DELETE FROM study_groups WHERE id = $1 RETURNING name;

-- name: ActiveAcademicYearLabel :one
SELECT label FROM academic_years WHERE is_active;

-- name: ListPublicClasses :many
SELECT c.code, c.name FROM classes c JOIN academic_years y ON y.id = c.academic_year_id
WHERE y.is_active AND c.is_active
ORDER BY lower(c.code), c.id;

-- name: ListPublicGroups :many
-- Only what a visitor may see. No email, no internal ids beyond what joins need.
SELECT g.name, g.updated_at, c.code AS class_code, t.code AS topic_code, t.title AS topic_title
FROM study_groups g
JOIN classes c ON c.id = g.class_id
JOIN academic_years y ON y.id = c.academic_year_id
JOIN topics t ON t.id = g.topic_id
WHERE y.is_active AND c.is_active AND g.is_active
  AND (sqlc.narg(class_code)::text IS NULL OR lower(c.code) = lower(sqlc.narg(class_code)::text))
  AND (sqlc.narg(query)::text IS NULL OR strpos(lower(g.name), lower(sqlc.narg(query)::text)) > 0)
ORDER BY lower(c.code), c.id, g.sort_order, lower(g.name), g.id
LIMIT 1000;
