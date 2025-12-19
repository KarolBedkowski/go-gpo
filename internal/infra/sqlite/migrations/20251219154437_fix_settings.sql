-- +goose Up
-- +goose StatementBegin

CREATE TABLE settings2 (
    user_id INTEGER NOT NULL,
    scope VARCHAR NOT NULL,
    podcast_id INTEGER REFERENCES podcasts(id) ON DELETE CASCADE ON UPDATE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE ON UPDATE CASCADE,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE ON UPDATE CASCADE,
    key VARCHAR NOT NULL,
    value VARCHAR NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

INSERT INTO settings2
SELECT user_id, scope, podcast_id, episode_id, device_id, key, value
FROM settings;

DROP TABLE settings;

ALTER TABLE settings2 RENAME TO settings;

CREATE UNIQUE INDEX settings_idx ON settings(user_id, scope, podcast_id, episode_id, device_id, key);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
