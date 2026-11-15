-- +goose Up
-- Progres Tugas Kelompok, part 3B: what the teacher records after checking
-- Gmail by hand. Nothing here is read from email and no file is stored; these
-- are the teacher's notes about each submission round (PLAN.md 6B/6D).

-- When any record of the group last changed (rounds, notebook). The public
-- "Terakhir Diperbarui" column uses the later of this and updated_at, so a
-- deleted round still moves the date forward.
ALTER TABLE study_groups ADD COLUMN progress_updated_at timestamptz;

-- One row per round: 0 = first submission, 1+ = revisions. A revision that
-- only carries a PPT never erases an earlier PDF; the submission lane is
-- computed from all rounds together.
CREATE TABLE submission_rounds (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id      bigint      NOT NULL REFERENCES study_groups (id) ON DELETE RESTRICT,
    round_number  integer     NOT NULL,
    received_at   timestamptz NOT NULL,
    sender_email  text,
    email_subject text        NOT NULL DEFAULT '',
    ppt_received  boolean     NOT NULL DEFAULT false,
    pdf_received  boolean     NOT NULL DEFAULT false,
    ppt_filename  text        NOT NULL DEFAULT '',
    pdf_filename  text        NOT NULL DEFAULT '',
    internal_note text        NOT NULL DEFAULT '',
    recorded_by   bigint      REFERENCES admin_users (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT submission_rounds_number_nonnegative CHECK (round_number >= 0),
    CONSTRAINT submission_rounds_has_file CHECK (ppt_received OR pdf_received),
    CONSTRAINT submission_rounds_ppt_name CHECK (ppt_received OR ppt_filename = ''),
    CONSTRAINT submission_rounds_pdf_name CHECK (pdf_received OR pdf_filename = ''),
    CONSTRAINT submission_rounds_sender_format CHECK (
        sender_email IS NULL OR (
            char_length(sender_email) <= 254
            AND sender_email ~ '^[a-z0-9._%+-]+@[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$'
        )
    ),
    CONSTRAINT submission_rounds_subject_safe CHECK (char_length(email_subject) <= 300 AND email_subject !~ '[[:cntrl:]]'),
    CONSTRAINT submission_rounds_filenames_safe CHECK (
        char_length(ppt_filename) <= 200 AND ppt_filename !~ '[/\\[:cntrl:]]'
        AND char_length(pdf_filename) <= 200 AND pdf_filename !~ '[/\\[:cntrl:]]'
    ),
    CONSTRAINT submission_rounds_note_length CHECK (char_length(internal_note) <= 2000)
);

-- Also serves lookups by group. A second insert of the same round number (a
-- double click or a retried request) fails here and becomes a 409.
CREATE UNIQUE INDEX submission_rounds_group_round_key ON submission_rounds (group_id, round_number);

-- Rounds are numbered without gaps: a new round must be the next number. The
-- group row is locked first so concurrent inserts for one group are counted
-- one after another.
-- +goose StatementBegin
CREATE FUNCTION submission_rounds_next_number() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    expected integer;
BEGIN
    PERFORM 1 FROM study_groups WHERE id = NEW.group_id FOR UPDATE;
    SELECT coalesce(max(round_number) + 1, 0) INTO expected
    FROM submission_rounds WHERE group_id = NEW.group_id;
    IF NEW.round_number <> expected THEN
        RAISE EXCEPTION 'round % is not the next round (%) for group %', NEW.round_number, expected, NEW.group_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER submission_rounds_next_number_before_insert
    BEFORE INSERT ON submission_rounds
    FOR EACH ROW EXECUTE FUNCTION submission_rounds_next_number();

-- One physical notebook per group, handed to the teacher in class.
CREATE TABLE group_notebooks (
    group_id    bigint      PRIMARY KEY REFERENCES study_groups (id) ON DELETE RESTRICT,
    is_received boolean     NOT NULL DEFAULT false,
    received_at timestamptz,
    note        text        NOT NULL DEFAULT '',
    updated_by  bigint      REFERENCES admin_users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT group_notebooks_received_time CHECK (is_received = (received_at IS NOT NULL)),
    CONSTRAINT group_notebooks_note_length CHECK (char_length(note) <= 1000)
);

-- +goose Down
DROP TABLE group_notebooks;
DROP TRIGGER submission_rounds_next_number_before_insert ON submission_rounds;
DROP FUNCTION submission_rounds_next_number();
DROP TABLE submission_rounds;
ALTER TABLE study_groups DROP COLUMN progress_updated_at;
