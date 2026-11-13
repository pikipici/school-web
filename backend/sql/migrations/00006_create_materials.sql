-- +goose Up
-- Plain-text learning materials. topic_id is optional: materials without a
-- topic are shown under "Umum". A topic in use cannot be deleted.
CREATE TABLE materials (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic_id     bigint REFERENCES topics (id) ON DELETE RESTRICT,
    title        text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'draft',
    sort_order   integer     NOT NULL DEFAULT 0,
    published_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT materials_title_length CHECK (char_length(btrim(title)) BETWEEN 1 AND 160 AND char_length(title) <= 160),
    CONSTRAINT materials_description_length CHECK (char_length(description) <= 20000),
    CONSTRAINT materials_status_known CHECK (status IN ('draft', 'published')),
    CONSTRAINT materials_published_has_time CHECK (status = 'draft' OR published_at IS NOT NULL),
    CONSTRAINT materials_sort_order_nonnegative CHECK (sort_order >= 0)
);

CREATE INDEX materials_topic_idx ON materials (topic_id);
CREATE INDEX materials_public_idx ON materials (topic_id, sort_order, id) WHERE status = 'published';

-- +goose Down
DROP TABLE materials;
