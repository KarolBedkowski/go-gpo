-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "users" (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE devices (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id INT NOT NULL,
    name VARCHAR NOT NULL,
    dev_type VARCHAR NOT NULL,
    caption VARCHAR,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE podcasts (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    url VARCHAR NOT NULL,
    subscribed BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata_updated_at TIMESTAMP WITH TIME ZONE,
    description TEXT,
    website TEXT,

    FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS "episodes" (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    podcast_id INTEGER NOT NULL,
    device_id INTEGER NULL,
    title VARCHAR NOT NULL,
    url VARCHAR NULL,
    action VARCHAR NOT NULL,
    started INTEGER,
    position INTEGER,
    total INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    guid VARCHAR,

    FOREIGN KEY (podcast_id) REFERENCES podcasts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS "settings" (
    user_id INTEGER NOT NULL,
    scope VARCHAR NOT NULL,
    podcast_id INTEGER REFERENCES podcasts(id) ON DELETE CASCADE ON UPDATE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE ON UPDATE CASCADE,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE ON UPDATE CASCADE,
    key VARCHAR NOT NULL,
    value VARCHAR NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE sessions (
    key VARCHAR NOT NULL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    data BYTEA
);

CREATE INDEX devices_user_id_idx ON devices (user_id);
CREATE INDEX podcasts_user_id_idx ON podcasts (user_id);
CREATE INDEX podcasts_idx1 ON podcasts (updated_at, subscribed);
CREATE INDEX podcasts_url ON podcasts (url);
CREATE INDEX episodes_device_id_idx ON episodes (device_id);
CREATE INDEX episodes_guid ON episodes (guid);
CREATE INDEX episodes_idx1 ON episodes (podcast_id, device_id, updated_at);
--CREATE INDEX episodes_idx2 ON episodes (podcast_id, url);
CREATE INDEX episodes_idx3 ON episodes USING btree (podcast_id, device_id, updated_at desc);
CREATE INDEX episodes_podcast_id_idx ON episodes (podcast_id);
CREATE INDEX expisode_action ON episodes (action);
CREATE INDEX expisode_updated_at ON episodes (updated_at);
CREATE INDEX expisode_url ON episodes (url);
CREATE INDEX podcasts_meta_updated_at ON podcasts (metadata_updated_at);
CREATE UNIQUE INDEX podcast_user_uniq ON podcasts (user_id, url);
CREATE UNIQUE INDEX settings_idx ON settings(user_id, scope, podcast_id, episode_id, device_id, key);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
