package model

//
// model_helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

func nvl(value ...string) string {
	for _, v := range value {
		if v != "" {
			return v
		}
	}

	return ""
}
