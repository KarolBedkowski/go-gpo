-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS episodes_idx2 ON episodes (podcast_id, url, updated_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS episodes_idx2;
-- +goose StatementEnd
