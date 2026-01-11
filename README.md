go-gpo 1.x
==========

Simple, self-hosted webserver that can handle podcast subscriptions for clients
that use gpodder protocol. Replacement and based on [mygpo] and [gpodder2go].

Implement most useful api's and tested with gPodder and AntennaPod.

[mygpo]: https://github.com/gpodder/mygpo
[gpodder2go]: https://github.com/oxtyped/gpodder2go


Features:
---------

 -  synchronize all user devices (no device synchronization api)
 -  support (partial) simple api and v2 api
 -  multi user
 -  single binary
 -  sqlite3 or PostgreSQL database
 -  simple (very!) web gui
 -  optionally download podcast and episodes metadata in configured interval (use
    only in gui).

Missing features from mygpo:

 -  simple api: toplist, suggestions, searching for podcast, podcasts lists (see
    below)
 -  advanced api
 -  device synchronization api (all user devices are synchronized)
 -  advanced & user-friendly webgui


Building and running
--------------------

### Dependency

 -  go 1.25+
 -  cgo
 -  [quicktemplate compiler qtc](github.com/valyala/quicktemplate/)

### Local Build

~~~~ shell
make
~~~~

or

~~~~ shell
go build -o go-gpo ./cli/
~~~~

### Run

Create/update database

~~~~ shell
./go-gpo database migrate
~~~~

Create user:

~~~~ shell
./go-gpo user add -u user1 -p passwordforuser1 -e email@of.user -n 'some user name'
~~~~

Run

~~~~  shell
./go-gpo serve
~~~~

Database configuration:

 -  Sqlite:

    ~~~~  shell
     ./go-gpo --db.driver=sqlite  --db.connstr='/some/path/database.sqlite' ...
    ~~~~

 -  PostgreSQL:

    ~~~~ shell
    ./go-gpo --db.driver=postgres  --db.connstr='host=127.0.0.1 user=gogpo password=xxxxx database=gogpo' ...
    ~~~~

    User and empty database must exists before run `database migrate`.

For other options / commands:

~~~~ shell
./go-gpo --help
~~~~

### Build tags

 -  `trace` - enable tracing (`/debug/requests`, `/debug/events` endpoints and
    additional data for `/debug/pprof/trace`); enable flight recorder


Implemented APIs
----------------

### Simple API

 -  [x] Downloading subscription lists
    `GET /subscriptions/{username}/{device_id}.{format}`
 -  [x] Uploading subscription lists
    `PUT /subscriptions/{username}/{device_id}.{format}`

### API v2

#### Authentication API

 -  [x] Login / Verify Login `POST /api/2/auth/(username)/login.json`
 -  [x] Logout `POST /api/2/auth/(username)/logout.json`

#### Device API

 -  [x] Update Device Data `POST /api/2/devices/(username)/(deviceid).json`
 -  [x] List Devices `GET /api/2/devices/(username).json`
 -  [x] Get Device Updates `GET /api/2/updates/(username)/(deviceid).json`

#### Subscriptions API

 -  [x] Get Subscriptions of Device `GET /subscriptions/(username)/(deviceid).(format)`
 -  [x] Get All Subscriptions `GET /subscriptions/(username).(format)`
 -  [x] Upload Subscriptions of Device
    `PUT /subscriptions/(username)/(deviceid).(format)`
 -  [x] Upload Subscription Changes
    `POST /api/2/subscriptions/(username)/(deviceid).json`
 -  [x] Get Subscription Changes `GET /api/2/subscriptions/(username)/(deviceid).json`

#### Episode Actions API

 -  [x] Upload Episode Actions `POST /api/2/episodes/(username).json`
 -  [x] Get Episode Actions `GET /api/2/episodes/(username).json`

#### Settings API

 -  [x] Save Settings `POST /api/2/settings/(username)/(scope).json`
 -  [x] Get Settings `GET /api/2/settings/(username)/(scope).json`

#### Favorites API

 -  [x] Get Favorite Episodes `GET /api/2/favorites/(username).json`


Not supported API
-----------------

### Simple API

 -  Downloading podcast toplists `GET /toplist/{number}.{format}`
 -  Downloading podcast suggestions `GET /suggestions/{number}.{format}`
 -  Searching for podcasts `GET /search.{format}?q={query}`

### Advanced API

 -  Add/remove subscriptions
    `POST /api/1/subscriptions/{username}/{device_id}.json`
 -  Retrieving subscription changes
    `GET /api/1/subscriptions/{username}/{device_id}.json?since={timestamp}`
 -  Uploading episode actions `POST /api/1/episodes/{username}.json`
 -  Retrieving episode actions `GET /api/1/episodes/{username}.json`
 -  (Re)naming devices and setting the type
    `POST /api/1/devices/{username}/{device-id}.json`
 -  Getting a list of devices `GET /api/1/devices/{username}.json`

### API v2

#### Device Synchronization API - not supported

 -  Get Sync Status `GET /api/2/sync-devices/(username).json`
 -  Start / Stop Sync `POST /api/2/sync-devices/(username).json`

#### Podcast Lists API - not supported

 -  Create Podcast List `/api/2/lists/{username}/create.{format}`
 -  Get User’s Lists `/api/2/lists/{username}.json`
 -  Get a Podcast List `/api/2/lists/{username}/list/{listname}.{format}`
 -  Update a Podcast List `/api/2/lists/{username}/list/{listname}.{format}`
 -  Delete a Podcast List `/api/2/lists/{username}/list/{listname}.{format}`


License
-------

Copyright (c) 2025-2026, Karol Będkowski.

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
