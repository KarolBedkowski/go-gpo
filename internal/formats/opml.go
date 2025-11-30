package formats

// opml.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"encoding/xml"
	"fmt"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

type Head struct {
	Title string `xml:"title"`
}
type Body struct {
	Outlines []Outline `xml:"outline"`
}

type Outline struct {
	Title  string `xml:"title,attr,omitempty"`
	Text   string `xml:"text,attr,omitempty"`
	Type   string `xml:"type,attr,omitempty"`
	XMLURL string `xml:"xmlUrl,attr,omitempty"`
}

// NewOPML creates a new OPML structure from a slice of bytes.
func NewOPML(title string) OPML {
	return OPML{
		Version: "2.0",
		Head: Head{
			Title: title,
		},
	}
}

func NewOPMLFromBytes(b []byte) (OPML, error) {
	var o OPML

	if err := xml.Unmarshal(b, &o); err != nil {
		return o, fmt.Errorf("unmarshal opml error: %w", err)
	}

	return o, nil
}

func (o *OPML) XML() ([]byte, error) {
	b, err := xml.MarshalIndent(o, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("marshal opml error: %w", err)
	}

	return append([]byte(xml.Header), b...), nil
}

func (o *OPML) AddRSS(url, title, text string) {
	outline := Outline{Type: "rss", XMLURL: url, Title: title, Text: text}
	o.Body.Outlines = append(o.Body.Outlines, outline)
}

func (o *OPML) AddURL(url ...string) {
	for _, u := range url {
		outline := Outline{Type: "rss", XMLURL: u}
		o.Body.Outlines = append(o.Body.Outlines, outline)
	}
}

func (o *OPML) ExtractsURLs() []string {
	subs := make([]string, 0)

	for _, i := range o.Body.Outlines {
		if url := i.XMLURL; url != "" {
			subs = append(subs, url)
		}
	}

	return subs
}
