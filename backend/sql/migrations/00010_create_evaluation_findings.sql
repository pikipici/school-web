-- +goose Up
ALTER TABLE study_groups ADD COLUMN review_status text NOT NULL DEFAULT 'BELUM_DICEK'
    CHECK (review_status IN ('BELUM_DICEK', 'DIPERIKSA', 'PERLU_REVISI', 'REVISI_DITERIMA', 'DISETUJUI'));
ALTER TABLE study_groups ADD COLUMN review_updated_at timestamptz NOT NULL DEFAULT now();

CREATE TABLE evaluation_findings (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id bigint NOT NULL REFERENCES study_groups(id) ON DELETE RESTRICT,
    file_type text NOT NULL CHECK (file_type IN ('PPT', 'PDF', 'BUKU', 'PRESENTASI')),
    location text NOT NULL CHECK (char_length(btrim(location)) BETWEEN 1 AND 200),
    problem text NOT NULL CHECK (char_length(btrim(problem)) BETWEEN 1 AND 2000),
    required_action text NOT NULL CHECK (char_length(btrim(required_action)) BETWEEN 1 AND 2000),
    priority text NOT NULL CHECK (priority IN ('WAJIB', 'SARAN')),
    resolved boolean NOT NULL DEFAULT false,
    published_at timestamptz,
    created_by bigint REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX evaluation_findings_group ON evaluation_findings(group_id, id);

-- +goose Down
DROP TABLE evaluation_findings;
ALTER TABLE study_groups DROP COLUMN review_updated_at, DROP COLUMN review_status;
