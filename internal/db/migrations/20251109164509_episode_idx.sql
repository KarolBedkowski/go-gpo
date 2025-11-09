-- +goose Up
-- +goose StatementBegin
CREATE INDEX expisode_action ON episodes (action);
CREATE INDEX expisode_updated_at ON episodes (updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX expisode_action;
DROP INDEX expisode_updated_at;
-- +goose StatementEnd
