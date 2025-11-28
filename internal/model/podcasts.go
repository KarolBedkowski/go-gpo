package model

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type Podcast struct {
	ID          int64
	Title       string
	URL         string
	Description string
	Subscribers int
	LogoURL     string
	Website     string
	MygpoLink   string
	Subscribed  bool
}

type PodcastWithLastEpisode struct {
	PodcastID   int64
	Title       string
	URL         string
	Description string
	Subscribers int
	LogoURL     string
	Website     string
	MygpoLink   string
	Subscribed  bool

	LastEpisode *Episode
}

func PodcastsToUrls(podcasts []Podcast) []string {
	urls := make([]string, len(podcasts))
	for i, p := range podcasts {
		urls[i] = p.URL
	}

	return urls
}
