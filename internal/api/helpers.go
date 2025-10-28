// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

// ensureList create empty list if `inp` is null or return `inp` otherwise.
func ensureList[T any](inp []T) []T {
	if inp == nil {
		return make([]T, 0)
	}

	return inp
}
