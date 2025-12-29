// Package model provide object used between api/web layer and services.
package model

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type ExportStruct struct {
	User     User
	Devices  []Device
	Podcasts Podcasts
	Episodes []Episode
	Settings []UserSettings
}

func (e *ExportStruct) FindEpisode(id int64) (Episode, bool) {
	for _, e := range e.Episodes {
		if e.ID == id {
			return e, true
		}
	}

	return Episode{}, false
}
