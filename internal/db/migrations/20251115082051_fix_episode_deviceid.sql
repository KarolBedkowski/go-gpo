-- +goose Up
-- +goose StatementBegin
CREATE TABLE episodes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    podcast_id INTEGER NOT NULL,
    device_id INTEGER NULL,
    title VARCHAR NOT NULL,
    url VARCHAR NULL,
    action VARCHAR NOT NULL,
    started INTEGER,
    position INTEGER,
    total INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, guid VARCHAR,

    FOREIGN KEY (podcast_id) REFERENCES podcasts (id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE SET NULL ON UPDATE CASCADE
);

INSERT INTO episodes_new
SELECT * FROM episodes;

DROP TABLE episodes;

ALTER TABLE episodes_new RENAME TO episodes;

CREATE INDEX episodes_device_id_idx ON episodes (device_id);
CREATE INDEX episodes_guid ON episodes(guid);
CREATE INDEX episodes_idx1 ON episodes (podcast_id, device_id, updated_at);
CREATE INDEX episodes_idx2 ON episodes (podcast_id, url);
CREATE INDEX episodes_podcast_id_idx ON episodes (podcast_id);
CREATE INDEX expisode_action ON episodes (action);
CREATE INDEX expisode_updated_at ON episodes (updated_at);
CREATE INDEX expisode_url ON episodes (url);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
