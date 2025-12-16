-- +goose Up
-- +goose StatementBegin

UPDATE podcasts SET updated_at = DATETIME() WHERE updated_at < '1970-01-01';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
