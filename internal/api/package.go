package api

//
// package.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(New),
	do.Lazy(newSimpleResource),
	do.Lazy(newAuthResource),
	do.Lazy(newDeviceResource),
	do.Lazy(newEpisodesResource),
	do.Lazy(newSettingsResource),
	do.Lazy(newSubscriptionsResource),
	do.Lazy(newUpdatesResource),
)
