package service

// package.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import "github.com/samber/do/v2"

//nolint:gochecknoglobals
var Package = do.Package(
	do.Lazy(NewUsersSrv),
	do.Lazy(NewDevicesSrv),
	do.Lazy(NewEpisodesSrv),
	do.Lazy(NewPodcastsSrv),
	do.Lazy(NewSettingsSrv),
	do.Lazy(NewSubscriptionsSrv),
	do.Lazy(NewMaintenanceSrv),
)
