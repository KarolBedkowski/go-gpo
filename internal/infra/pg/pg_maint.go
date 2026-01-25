package pg

//
// sqlite_maint.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"embed"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
)

//go:embed "migrations/*.sql"
var embedMigrations embed.FS

func (Repository) Maintenance(ctx context.Context) error {
	logger := log.Ctx(ctx)
	dbi := db.MustCtx(ctx)

	for idx, sql := range maintScripts {
		logger.Debug().Msgf("pg.Repository: run maintenance script=%d sql=%q", idx, sql)

		res, err := dbi.ExecContext(ctx, sql)
		if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance script failed").
				WithMeta("sql", sql)
		}

		rowsaffected, err := res.RowsAffected()
		if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance script - failed get rows affected").
				WithMeta("sql", sql)
		}

		logger.Debug().Msgf("pg.Repository: run maintenance script=%d finished; affected=%d", idx, rowsaffected)
	}

	// print some stats
	var numEpisodes, numPodcasts, numEpisodeHist int
	if err := dbi.GetContext(ctx, &numEpisodes, "SELECT count(*) FROM episodes"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count episodes failed")
	}

	if err := dbi.GetContext(ctx, &numPodcasts, "SELECT count(*) FROM podcasts"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count podcasts failed")
	}

	if err := dbi.GetContext(ctx, &numEpisodeHist, "SELECT count(*) FROM episodes_hist"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count episodes hist failed")
	}

	logger.Info().Msgf("pg.Repository: database maintenance finished; podcasts=%d; episodes=%d, episodes_hist=%d",
		numPodcasts, numEpisodes, numEpisodeHist)

	return nil
}

//------------------------------------------------------------------------------

//nolint:gochecknoglobals
var maintScripts = []string{
	// delete play actions when for given episode never play action exists
	`
	DELETE FROM episodes_hist AS e
	WHERE action = 'play'
		AND updated_at < now() - INTERVAL '14 day'
		AND EXISTS (
			SELECT NULL FROM episodes_hist AS eh
			WHERE eh.episode_id  = e.episode_id AND eh.action = 'play' AND eh.updated_at > e.updated_at
		);
	`,
}
