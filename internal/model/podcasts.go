package model

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type Podcast struct {
	Title       string
	URL         string
	Description string
	Subscribers int
	LogoURL     string
	Website     string
	MygpoLink   string
}

type PodcastWithLastEpisode struct {
	Title       string
	URL         string
	Description string
	Subscribers int
	LogoURL     string
	Website     string
	MygpoLink   string

	LastEpisode *Episode
}

func PodcastsToUrls(podcasts []Podcast) []string {
	urls := make([]string, len(podcasts))
	for i, p := range podcasts {
		urls[i] = p.URL
	}

	return urls
}
