package validators

//
// validators.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import "slices"

var ValidDevTypes = []string{"desktop", "laptop", "mobile", "server", "other"}

func IsValidDevType(deviceType string) bool {
	return slices.Contains(ValidDevTypes, deviceType)
}
