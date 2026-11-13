-- +goose Up
-- PDF/PPT/PPTX files attached to a material by the teacher (PLAN.md section 0).
-- The file lives on disk under a server-chosen UUID name (storage_key); the
-- original name is display text only and can never be used as a path.
CREATE TABLE material_attachments (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    material_id   bigint      NOT NULL REFERENCES materials (id) ON DELETE CASCADE,
    storage_key   uuid        NOT NULL,
    original_name text        NOT NULL,
    kind          text        NOT NULL,
    content_type  text        NOT NULL,
    size_bytes    bigint      NOT NULL,
    sha256        bytea       NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT material_attachments_kind_known CHECK (
        (kind = 'pdf'  AND content_type = 'application/pdf') OR
        (kind = 'ppt'  AND content_type = 'application/vnd.ms-powerpoint') OR
        (kind = 'pptx' AND content_type = 'application/vnd.openxmlformats-officedocument.presentationml.presentation')
    ),
    CONSTRAINT material_attachments_size_range CHECK (size_bytes BETWEEN 1 AND 26214400),
    CONSTRAINT material_attachments_sha256_length CHECK (octet_length(sha256) = 32),
    CONSTRAINT material_attachments_name_safe CHECK (
        char_length(btrim(original_name)) >= 1
        AND char_length(original_name) <= 200
        AND original_name !~ '[/\\[:cntrl:]]'
    )
);

CREATE UNIQUE INDEX material_attachments_storage_key_key ON material_attachments (storage_key);
CREATE INDEX material_attachments_material_idx ON material_attachments (material_id, id);

-- At most five attachments per material. The parent row is locked first, so
-- concurrent uploads to the same material are counted one after another.
-- +goose StatementBegin
CREATE FUNCTION material_attachments_limit() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    n integer;
BEGIN
    PERFORM 1 FROM materials WHERE id = NEW.material_id FOR UPDATE;
    SELECT count(*) INTO n FROM material_attachments WHERE material_id = NEW.material_id;
    IF n >= 5 THEN
        RAISE EXCEPTION 'material % already has 5 attachments', NEW.material_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER material_attachments_limit_before_insert
    BEFORE INSERT ON material_attachments
    FOR EACH ROW EXECUTE FUNCTION material_attachments_limit();

-- +goose Down
DROP TRIGGER material_attachments_limit_before_insert ON material_attachments;
DROP FUNCTION material_attachments_limit();
DROP TABLE material_attachments;
