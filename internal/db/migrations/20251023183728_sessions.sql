-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    key VARCHAR NOT NULL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    data BLOB
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd
