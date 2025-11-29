-- +goose Up
-- +goose StatementBegin

ALTER TABLE podcasts ADD metadata_updated_at DATETIME;
ALTER TABLE podcasts ADD description TEXT;
ALTER TABLE podcasts ADD website TEXT;

CREATE INDEX podcasts_meta_updated_at ON podcasts (metadata_updated_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE podcasts DROP metadata_updated_at;
ALTER TABLE podcasts DROP description;
ALTER TABLE podcasts DROP website;
-- +goose StatementEnd
