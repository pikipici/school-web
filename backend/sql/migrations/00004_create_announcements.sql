-- +goose Up
-- Plain-text announcements. Only rows with status 'published' may ever be
-- returned by public endpoints; published_at is set on first publish.
CREATE TABLE announcements (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title        text        NOT NULL,
    body         text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'draft',
    pinned       boolean     NOT NULL DEFAULT false,
    published_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT announcements_title_length CHECK (char_length(btrim(title)) BETWEEN 1 AND 160 AND char_length(title) <= 160),
    CONSTRAINT announcements_body_length CHECK (char_length(body) <= 10000),
    CONSTRAINT announcements_status_known CHECK (status IN ('draft', 'published')),
    CONSTRAINT announcements_published_has_time CHECK (status = 'draft' OR published_at IS NOT NULL)
);

-- Public listing: published, pinned first, newest first.
CREATE INDEX announcements_public_idx ON announcements (pinned DESC, published_at DESC) WHERE status = 'published';

-- +goose Down
DROP TABLE announcements;
