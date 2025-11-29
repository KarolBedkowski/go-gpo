-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings (
    user_id INT NOT NULL,
    scope VARCHAR NOT NULL,
    key VARCHAR NOT NULL,
    value VARCHAR NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX settings_idx ON settings(user_id, scope, key);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE settings;
-- +goose StatementEnd
