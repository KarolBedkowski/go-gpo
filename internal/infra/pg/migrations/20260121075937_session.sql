-- +goose Up
-- +goose NO TRANSACTION
-- +goose StatementBegin
ALTER TABLE sessions SET UNLOGGED;
-- +goose StatementEnd

-- +goose Down
-- +goose NO TRANSACTION
-- +goose StatementBegin
ALTER TABLE sessions SET LOGGED;
-- +goose StatementEnd
