-- +goose Up
-- +goose NO TRANSACTION
-- +goose StatementBegin

PRAGMA foreign_keys=OFF;

CREATE TABLE users2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users2
SELECT id, username, password, email, name, created_at, updated_at FROM users;

DROP TABLE users;

ALTER TABLE users2 RENAME TO users;


PRAGMA foreign_keys=ON;


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
