package web

//
// package.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/web/templates"
)

//nolint:gochecknoglobals
var Package = do.Package(
	do.Lazy(New),
	do.Lazy(newDevicePages),
	do.Lazy(newEpisodePages),
	do.Lazy(newPodcastPages),
	do.Lazy(newUserPages),
	do.Lazy(newIndexPage),
	do.Lazy(templates.NewRenderer),
)
