//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package validators

import (
	"net/url"
	"strings"
)

// ------------------------------------------------------

// SanitizeURLs remove non-http* urls; correct others. Return list of "safe" urls and list of changes
// in url [[old url, corrected url]].
func SanitizeURLs(urls []string) ([]string, [][]string) {
	res := make([]string, 0, len(urls))
	changes := make([][]string, 0)

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

// SanitizeURL normalize given url.
// Based on mygpo; but do not normalize query & path; do not expand shortcuts, remove user/pass.
// Accept only http/s.
func SanitizeURL(u string) string {
	su := strings.TrimSpace(u)

	// like mygpo
	if len(su) < 8 { //nolint:mnd
		return ""
	}

	purl, err := url.Parse(su)
	if err != nil {
		return ""
	}

	// url without scheme are http; feed://, itpc:// and itms:// are really http://
	if purl.Scheme == "" || purl.Scheme == "feed" || purl.Scheme == "itpc" || purl.Scheme == "itms" {
		purl.Scheme = "http"
	}

	// scheme and host are case insensitive
	purl.Scheme = strings.ToLower(purl.Scheme)
	purl.Host = strings.ToLower(purl.Host)

	// Normalize empty paths to "/"
	if purl.Path == "" {
		purl.Path = "/"
	}

	// accept only http & https
	if purl.Scheme != "http" && purl.Scheme != "https" {
		return ""
	}

	return purl.String()
}
