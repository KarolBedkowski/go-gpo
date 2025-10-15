/*
 * schema.sql
 * Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
 *
 * Distributed under terms of the GPLv3 license.
 */

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP default current_timestamp,
    updated_at TIMESTAMP default current_timestamp
);


CREATE TABLE devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INT NOT NULL,
    name VARCHAR NOT NULL,
    dev_type VARCHAR NOT NULL,
    caption VARCHAR,
    created_at TIMESTAMP default current_timestamp,
    updated_at TIMESTAMP default current_timestamp,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INT NOT NULL,
    podcast VARCHAR NOT NULL,
    action VARCHAR NOT NULL,
    ts TIMESTAMP,
    created_at TIMESTAMP default current_timestamp,
    updated_at TIMESTAMP default current_timestamp,

    FOREIGN KEY (device_id) REFERENCES devices (id)
);


CREATE TABLE sync_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    sync_status VARCHAR NOT NULL DEFAULT 'pending',
    sync_time TIMESTAMP NOT NULL DEFAULT current_timestamp,
    created_at TIMESTAMP DEFAULT current_timestamp,
    updated_at TIMESTAMP DEFAULT current_timestamp,

    FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE sync_group_devices (
    sync_group_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,

    FOREIGN KEY (device_id) REFERENCES devices (id),
    FOREIGN KEY (sync_group_id) REFERENCES sync_groups (id)
);


CREATE TABLE podcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title varchar NOT NULL,
    subtitle varchar,
    description varchar,
    link varchar,
    last_update TIMESTAMP,
    created_at TIMESTAMP DEFAULT current_timestamp,
    updated_at TIMESTAMP DEFAULT current_timestamp,
    latest_episode_ts TIMESTAMP
);

CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title varchar NOT NULL,
    subtitle varchar,
    description varchar,
    link varchar NULL,
    last_update TIMESTAMP,
    created_at TIMESTAMP DEFAULT current_timestamp,
    updated_at TIMESTAMP DEFAULT current_timestamp,
    released TIMESTAMP,
    podcast_id INTEGER NOT NULL,

    FOREIGN KEY (podcast_id) REFERENCES podcasts (id)
);


CREATE TABLE history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts TIMESTAMP DEFAULT current_timestamp,
    action varchar NOT NULL,
    device_id INTEGER NULL,
    podcast_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (device_id) REFERENCES devices (id),
    FOREIGN KEY (podcast_id) REFERENCES podcasts (id)
);

CREATE TABLE episodes_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts TIMESTAMP DEFAULT current_timestamp,
    created_at TIMESTAMP DEFAULT current_timestamp,
	action varchar NOT NULL,
	started INTEGER,
	stopped INTEGER.
	total INTEGER,
	user_id INTEGER NOT NULL,
	device_id INTEGER,
	episode_id INTEGER,

    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (device_id) REFERENCES devices (id),
    FOREIGN KEY (episode_id) REFERENCES episodes (id)
);

CREATE TABLE public.episodestates_episodestate (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ts TIMESTAMP DEFAULT current_timestamp,
    created_at TIMESTAMP DEFAULT current_timestamp,
	action varchar NOT NULL,

	user_id INTEGER NOT NULL,
	episode_id INTEGER,

    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (episode_id) REFERENCES episodes (id)
);



insert into users (username, password, email, name) values ('k', 'k', 'k@localhost', 'k');




-- vim:et
