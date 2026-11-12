-- +goose Up
-- Append-only trail of security-relevant actions. actor_user_id is deliberately
-- not a foreign key so history survives deleting the actor. Never store
-- passwords, tokens, or free-form user input in detail.
CREATE TABLE audit_log (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    actor_user_id bigint,
    action        text        NOT NULL,
    outcome       text        NOT NULL,
    ip            inet,
    detail        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT audit_log_action_format CHECK (action ~ '^[a-z][a-z0-9_.]{1,63}$'),
    CONSTRAINT audit_log_outcome_known CHECK (outcome IN ('success', 'failure'))
);

CREATE INDEX audit_log_occurred_at_idx ON audit_log (occurred_at DESC);

-- +goose StatementBegin
CREATE FUNCTION audit_log_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_log_no_update_delete
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_reject_change();

-- +goose Down
DROP TRIGGER audit_log_no_update_delete ON audit_log;
DROP FUNCTION audit_log_reject_change();
DROP TABLE audit_log;
