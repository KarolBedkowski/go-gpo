package templates

//
// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"strconv"
	"time"
)

func formatDateTime(t time.Time) string {
	return t.Format(time.DateTime)
}

func formatPInt32(v *int32) string {
	if v == nil {
		return ""
	}

	return strconv.Itoa(int(*v))
}

func formatPInt32AsDuration(v *int32) string {
	if v == nil {
		return ""
	}

	return (time.Duration(int(*v)) * time.Second).String()
}
