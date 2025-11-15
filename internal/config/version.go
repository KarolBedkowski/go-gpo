package config

//
// version.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"runtime/debug"
)

var (
	Version   = "dev"
	Revision  = ""
	BuildDate = ""
	BuildUser = ""
	Branch    = ""

	VersionString = ""
)

func init() { //nolint:gochecknoinits
	VersionString = Version

	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			var dirty string

			for _, kv := range info.Settings {
				switch kv.Key {
				case "vcs.revision":
					Revision = kv.Value
				case "vcs.time":
					BuildDate = kv.Value
				case "vcs.modified":
					dirty = kv.Value
				}
			}

			VersionString = fmt.Sprintf("Rev: %s at %s %s", Revision, BuildDate, dirty)
		}
	} else {
		VersionString = fmt.Sprintf("Ver: %s, Rev: %s, Build: %s by %s from %s",
			Version, Revision, BuildDate, BuildUser, Branch)
	}
}
