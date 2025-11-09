-- +goose Up
-- +goose StatementBegin

DROP INDEX settings_idx;

ALTER TABLE settings ADD COLUMN podcast_id INT REFERENCES podcasts(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE settings ADD COLUMN episode_id INT REFERENCES episodes(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE settings ADD COLUMN device_id INT REFERENCES devices(id) ON DELETE CASCADE ON UPDATE CASCADE;

CREATE UNIQUE INDEX settings_idx ON settings(user_id, scope, podcast_id, episode_id, device_id, key);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX settings_idx;

ALTER TABLE settings DROP COLUMN podcast_id;
ALTER TABLE settings DROP COLUMN episode_id;
ALTER TABLE settings DROP COLUMN device_id;

CREATE UNIQUE INDEX settings_idx ON settings(user_id, scope, key);

-- +goose StatementEnd
