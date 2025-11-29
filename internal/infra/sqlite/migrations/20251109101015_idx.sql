-- +goose Up
-- +goose StatementBegin
CREATE INDEX expisode_url ON episodes (url);
CREATE INDEX podcasts_url ON podcasts (url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX expisode_url;
DROP INDEX podcasts_url;
-- +goose StatementEnd
