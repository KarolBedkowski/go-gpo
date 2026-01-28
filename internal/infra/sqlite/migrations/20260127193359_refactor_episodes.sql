-- +goose Up
-- +goose StatementBegin

CREATE TABLE episodes_hist (
	episode_id int8 NOT NULL,
	device_id int8 NULL,
	"action" varchar NOT NULL,
	started int4 NULL,
	"position" int4 NULL,
	total int4 NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT episodes_hist_device_id_fkey FOREIGN KEY (device_id)
	    REFERENCES devices(id)
	    ON DELETE SET NULL ON UPDATE CASCADE,
	CONSTRAINT episodes_hist_episode_id_fkey FOREIGN KEY (episode_id)
	    REFERENCES episodes(id)
	    ON DELETE CASCADE ON UPDATE CASCADE
);



INSERT INTO episodes_hist
	(episode_id, device_id, "action", started, "position", total, created_at, updated_at)
SELECT (
		SELECT id
		FROM episodes e2
		WHERE e2.url = e.url AND e2.podcast_id = e.podcast_id
		ORDER BY e2.updated_at DESC
		LIMIT 1
	) AS id,
	device_id, "action", started, "position", total, created_at, updated_at
FROM episodes as e;

CREATE INDEX episodes_hist_device_id_idx ON episodes_hist(device_id);
CREATE INDEX episodes_hist_epiosde_id_idx ON episodes_hist(episode_id);
CREATE INDEX episodes_hist_idx1 ON episodes_hist(episode_id, updated_at);
CREATE INDEX episodes_hist_updatedat_idx ON episodes_hist(updated_at);

DELETE FROM episodes as e
WHERE NOT exists (
	SELECT NULL FROM episodes_hist eh WHERE eh.episode_id = e.id
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

INSERT INTO episodes
	(podcast_id, device_id, title, url, "action", started, "position", total, created_at, updated_at, guid)
SELECT e.podcast_id, eh.device_id, e.title, e.url,
	eh."action", eh.started, eh."position", eh.total, eh.created_at, eh.updated_at,
	e.guid
FROM episodes_hist as eh
JOIN episodes as e ON e.id = eh.episode_id
WHERE NOT EXISTS (
	SELECT NULL
	FROM episodes as e2
	WHERE e2.url = e.url AND e2.podcast_id = e.podcast_id  AND e2.updated_at = eh.updated_at
);


DROP TABLE episodes_hist;

-- +goose StatementEnd
