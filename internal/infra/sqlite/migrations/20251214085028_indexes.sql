-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS podcast_user_uniq ON podcasts (user_id, url);

PRAGMA OPTIMIZE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS podcast_user_uniq;
-- +goose StatementEnd
