package service

import (
	"net/url"
	"strings"
)

//
// sanitize.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

func SanitizeURLs(urls []string) ([]string, [][]string) {
	res := make([]string, 0, len(urls))
	var changes [][]string

	for _, u := range urls {
		su := SanitizeURL(u)

		if su == "" {
			continue
		}

		if su != u {
			changes = append(changes, []string{u, su})
		}

		res = append(res, su)
	}
	return res, changes
}

func SanitizeURL(u string) string {
	su := strings.TrimSpace(u)
	url, err := url.Parse(su)
	if err != nil || (url.Scheme != "http" && url.Scheme != "https") {
		return ""
	}

	return su
}
