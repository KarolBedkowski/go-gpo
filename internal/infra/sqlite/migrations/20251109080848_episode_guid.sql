-- +goose Up
-- +goose StatementBegin
ALTER TABLE episodes ADD COLUMN guid VARCHAR;

CREATE INDEX episodes_guid ON episodes(guid);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX episodes_guid;

ALTER TABLE episodes DROP COLUMN guid;
-- +goose StatementEnd
