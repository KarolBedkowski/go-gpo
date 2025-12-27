package validators

import "slices"

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

//nolint:gochecknoglobals
var ValidActions = []string{"download", "delete", "play", "new", "flattr", ""}

func IsValidEpisodeAction(action string) bool {
	return slices.Contains(ValidActions, action)
}
