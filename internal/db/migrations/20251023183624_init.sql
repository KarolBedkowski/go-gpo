-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=ON;

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INT NOT NULL,
    name VARCHAR NOT NULL,
    dev_type VARCHAR NOT NULL,
    caption VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX devices_user_id_idx ON devices (user_id);


CREATE TABLE podcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    url VARCHAR NOT NULL,
    subscribed BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX podcasts_user_idx2 ON podcasts (user_id, url);
CREATE INDEX podcasts_user_id_idx ON podcasts (user_id);
CREATE INDEX podcasts_idx1 ON podcasts (updated_at, subscribed);


CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    podcast_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    url VARCHAR NULL,
    action VARCHAR NOT NULL,
    started INTEGER,
    position INTEGER,
    total INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (podcast_id) REFERENCES podcasts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX episodes_device_id_idx ON episodes (device_id);
CREATE INDEX episodes_podcast_id_idx ON episodes (podcast_id);
CREATE INDEX episodes_idx1 ON episodes (podcast_id, device_id, updated_at);
CREATE UNIQUE INDEX episodes_idx2 ON episodes (podcast_id, url);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
