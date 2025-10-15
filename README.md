# go-gpodder

Reimplementatin (simpilfed) (mygpo)[https://github.com/gpodder/mygpo] and (gpodder2go)[https://github.com/oxtyped/gpodder2go].

Features:

* synchronize all user devices (no device synchronization api)
* support (partial) simple api and v2 api
* multi user
* single binary
* sqlite3 database

Missing features from mygpo:

* simple api: toplist, suggestions, searching for podcast
* advanced api
* favorites api
* device synchronization api (all user devices are synchronized)
* webgui

## Building and running

### Dependency

* go 1.25+
* cgo

### Local Build

    make

or

    go build -o go-gpodder ./cli/

### Run

Create/update database

    ./go-gpodder migrate

Run

    ./go-gpodder serve


For other options / commands:

    ./go-gpodder --help


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
