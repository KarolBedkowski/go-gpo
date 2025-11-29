-- +goose Up
-- +goose StatementBegin
ALTER TABLE devices ADD COLUMN last_seen_at TIMESTAMP;

UPDATE devices SET last_seen_at = datetime(updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE devices DROP COLUMN last_seen_at;
-- +goose StatementEnd
