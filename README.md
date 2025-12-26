# go-gpo

Simple, self-hosted webserver that can handle podcast subscriptions for clients that use gpodder protocol.
Replacement and based on [mygpo](https://github.com/gpodder/mygpo) and [gpodder2go](https://github.com/oxtyped/gpodder2go).

Implement most useful api's and tested with gPodder and AntennaPod.

## Features:

* synchronize all user devices (no device synchronization api)
* support (partial) simple api and v2 api
* multi user
* single binary
* sqlite3 database
* simple (very!) web gui
* optionally download podcast and episodes metadata in configured interval (use only in gui).

Missing features from mygpo:

* simple api: toplist, suggestions, searching for podcast, podcasts lists (see below)
* advanced api
* device synchronization api (all user devices are synchronized)
* advanced & user-friendly webgui


## Building and running

### Dependency

* go 1.25+
* cgo


### Local Build

    make

or

    go build -o go-gpo ./cli/


### Run

Create/update database

    ./go-gpo database migrate

Run

    ./go-gpo serve

Configure database file:

    ./go-gpo --database='/some/path/database.sqlite?_fk=1&_journal_mode=WAL&_synchronous=NORMAL' ...

For other options / commands:

    ./go-gpo --help


## Implemented APIs

### Simple API

* [x] Downloading subscription lists `GET /subscriptions/{username}/{device_id}.{format}`
* [x] Uploading subscription lists `PUT /subscriptions/{username}/{device_id}.{format}`
* [-] Downloading podcast toplists `GET /toplist/{number}.{format}`
* [-] Downloading podcast suggestions `GET /suggestions/{number}.{format}`
* [-] Searching for podcasts `GET /search.{format}?q={query}`

### Advanced API

* [?] Add/remove subscriptions `POST /api/1/subscriptions/{username}/{device_id}.json`
* [?] Retrieving subscription changes `GET /api/1/subscriptions/{username}/{device_id}.json?since={timestamp}`
* [?] Uploading episode actions `POST /api/1/episodes/{username}.json`
* [?] Retrieving episode actions `GET /api/1/episodes/{username}.json`
* [?] (Re)naming devices and setting the type `POST /api/1/devices/{username}/{device-id}.json`
* [?] Getting a list of devices `GET /api/1/devices/{username}.json`

### API v2

#### Authentication API

* [x] Login / Verify Login `POST /api/2/auth/(username)/login.json`
* [x] Logout `POST /api/2/auth/(username)/logout.json`

#### Device API

* [x] Update Device Data `POST /api/2/devices/(username)/(deviceid).json`
* [x] List Devices [DONE] `GET /api/2/devices/(username).json`
* [x] Get Device Updates `GET /api/2/updates/(username)/(deviceid).json`

#### Subscriptions API

* [x] Get Subscriptions of Device `GET /subscriptions/(username)/(deviceid).(format)`
* [x] Get All Subscriptions `GET /subscriptions/(username).(format)`
* [x] Upload Subscriptions of Device `PUT /subscriptions/(username)/(deviceid).(format)`
* [x] Upload Subscription Changes `POST /api/2/subscriptions/(username)/(deviceid).json`
* [x] Get Subscription Changes `GET /api/2/subscriptions/(username)/(deviceid).json`

#### Episode Actions API

* [x] Upload Episode Actions `POST /api/2/episodes/(username).json`
* [x] Get Episode Actions `GET /api/2/episodes/(username).json`

#### Settings API

* [x] Save Settings `POST /api/2/settings/(username)/(scope).json`
* [x] Get Settings `GET /api/2/settings/(username)/(scope).json`

#### Favorites API

* [X] Get Favorite Episodes `GET /api/2/favorites/(username).json`

#### Device Synchronization API

* [-] Get Sync Status `GET /api/2/sync-devices/(username).json`
* [-] Start / Stop Sync `POST /api/2/sync-devices/(username).json`

#### Podcast Lists API
* [-] Create Podcast List `/api/2/lists/{username}/create.{format}`
* [-] Get User’s Lists `/api/2/lists/{username}.json`
* [-] Get a Podcast List `/api/2/lists/{username}/list/{listname}.{format}`
* [-] Update a Podcast List `/api/2/lists/{username}/list/{listname}.{format}`
* [-] Delete a Podcast List `/api/2/lists/{username}/list/{listname}.{format}`


All devices for one account are always synchronized.

##  License

Copyright (c) 2025, Karol Będkowski.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see http://www.gnu.org/licenses/.
