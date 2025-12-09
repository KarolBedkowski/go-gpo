package validators

import "regexp"

//
// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

var reUserName = regexp.MustCompile(`^[\w+.-]+$`)

func IsValidUserName(name string) bool {
	return reUserName.MatchString(name)
}
