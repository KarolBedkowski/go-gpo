//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//---------------------------------------------------------------------

func newDownloadPodcastsInfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "download-info",
		Usage: "download podcast metadata",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "max-age",
				Usage: "max age of existing podcast metadata to update",
			},
		},
		Action: wrap(downloadPodcastsInfoCmd),
	}
}

func downloadPodcastsInfoCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	podcastSrv := do.MustInvoke[*service.PodcastsSrv](injector)

	maxAge := time.Now().UTC()
	if since := clicmd.Duration("max-age"); since > 0 {
		maxAge = maxAge.Add(-since)
	}

	if err := podcastSrv.DownloadPodcastsInfo(ctx, maxAge); err != nil {
		return fmt.Errorf("download podcast info failed: %w", err)
	}

	fmt.Println("Podcast info downloaded")

	return nil
}
