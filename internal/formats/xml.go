package formats

//
// xml.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import "gitlab.com/kabes/go-gpo/internal/model"

// ------------------------------------------------------

type XMLPodcasts struct {
	Podcasts xmlPodcasts `xml:"podcasts"`
}

type xmlPodcasts struct {
	Podcasts []xmlPodcast `xml:"podcast"`
}

type xmlPodcast struct {
	Title         string `xml:"title"`
	URL           string `xml:"url"`
	Website       string `xml:"website"`
	MygpoLink     string `xml:"mygpo_link"`
	Author        string `xml:"author"`
	Description   string `xml:"description"`
	Subscribers   int    `xml:"subscribers"`
	LogoURL       string `xml:"logo_url"`
	ScaledLogoURL string `xml:"scaled_logo_url"`
}

func NewXMLPodcasts(podcasts []model.Podcast) XMLPodcasts {
	xmlpod := make([]xmlPodcast, len(podcasts))

	for i, p := range podcasts {
		xmlpod[i] = xmlPodcast{
			Title:         p.Title,
			URL:           p.URL,
			Website:       p.Website,
			MygpoLink:     p.MygpoLink,
			Author:        "",
			Description:   p.Description,
			Subscribers:   0,
			LogoURL:       "",
			ScaledLogoURL: "",
		}
	}

	return XMLPodcasts{
		Podcasts: xmlPodcasts{
			Podcasts: xmlpod,
		},
	}
}
