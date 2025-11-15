package model

//
// model_helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

// Coalesce return first not empty string from arguments.
func Coalesce(value ...string) string {
	for _, v := range value {
		if v != "" {
			return v
		}
	}

	return ""
}

// NVL return *value if value != nil or T otherwise.
func NVL[T any](value *T, def T) T {
	if value == nil {
		return def
	}

	return *value
}

// NilIf return nil when value is equal to def or pointer to value otherwise.
func NilIf(value, def string) *string {
	if value == def {
		return nil
	}

	return &value
}

func Map[T, R any](collection []T, iteratee func(item *T) R) []R {
	result := make([]R, len(collection))

	for i := range collection {
		result[i] = iteratee(&collection[i])
	}

	return result
}
