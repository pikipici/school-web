-- +goose Up
-- Progres Tugas Kelompok, part 3A: academic years, classes, and study groups.
-- No student names are stored (PHASE3_PLAN K5). The group email is for the
-- teacher only and is never returned by public or group endpoints.

CREATE TABLE academic_years (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label      text        NOT NULL,
    is_active  boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT academic_years_label_format CHECK (
        CASE WHEN label ~ '^[0-9]{4}/[0-9]{4}$'
             THEN substr(label, 6, 4)::int = substr(label, 1, 4)::int + 1
             ELSE false END
    )
);
CREATE UNIQUE INDEX academic_years_label_key ON academic_years (label);
-- At most one active year; the Go service keeps at least one.
CREATE UNIQUE INDEX academic_years_one_active ON academic_years (is_active) WHERE is_active;

INSERT INTO academic_years (label, is_active) VALUES ('2026/2027', true);

CREATE TABLE classes (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    academic_year_id bigint      NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
    code             text        NOT NULL,
    name             text        NOT NULL DEFAULT '',
    is_active        boolean     NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT classes_code_format CHECK (
        char_length(code) BETWEEN 1 AND 20
        AND code ~ '^[0-9A-Za-z][0-9A-Za-z .-]*$'
    ),
    CONSTRAINT classes_name_length CHECK (char_length(name) <= 80 AND name !~ '[[:cntrl:]]')
);
CREATE UNIQUE INDEX classes_year_code_key ON classes (academic_year_id, lower(code));

CREATE TABLE study_groups (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    class_id       bigint      NOT NULL REFERENCES classes (id) ON DELETE RESTRICT,
    topic_id       bigint      NOT NULL REFERENCES topics (id) ON DELETE RESTRICT,
    name           text        NOT NULL,
    expected_email text,
    sort_order     integer     NOT NULL DEFAULT 0,
    is_active      boolean     NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT study_groups_name_length CHECK (
        char_length(btrim(name)) >= 1 AND char_length(name) <= 60 AND name !~ '[[:cntrl:]]'
    ),
    CONSTRAINT study_groups_email_format CHECK (
        expected_email IS NULL OR (
            char_length(expected_email) <= 254
            AND expected_email ~ '^[a-z0-9._%+-]+@[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$'
        )
    ),
    CONSTRAINT study_groups_sort_order_nonnegative CHECK (sort_order >= 0)
);
CREATE UNIQUE INDEX study_groups_class_name_key ON study_groups (class_id, lower(name));
CREATE INDEX study_groups_topic_idx ON study_groups (topic_id);

-- +goose Down
DROP TABLE study_groups;
DROP TABLE classes;
DROP TABLE academic_years;
