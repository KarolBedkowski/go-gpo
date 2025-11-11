-- +goose Up
-- +goose StatementBegin
-- drop unique index
DROP INDEX IF EXISTS episodes_idx2;
-- create not-unique index
CREATE INDEX episodes_idx2 ON episodes (podcast_id, url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
