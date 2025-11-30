package api

//
// api_simple_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"testing"

	"gitlab.com/kabes/go-gpo/internal/assert"
)

func TestFormatOPML(t *testing.T) {
	urls := []string{
		"http://www.example.com/podcast1/podcast.xml",
		"http://www.example.com/podcast2/podcast.xml",
		"http://www.example.com/podcast3/podcast.xml",
	}
	exp := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
	<head>
		<title>go-gpo</title>
	</head>
	<body>
		<outline type="rss" xmlUrl="http://www.example.com/podcast1/podcast.xml"></outline>
		<outline type="rss" xmlUrl="http://www.example.com/podcast2/podcast.xml"></outline>
		<outline type="rss" xmlUrl="http://www.example.com/podcast3/podcast.xml"></outline>
	</body>
</opml>`

	res, err := formatOMPL(urls)
	assert.NoErr(t, err)
	assert.Equal(t, string(res), exp)
}
