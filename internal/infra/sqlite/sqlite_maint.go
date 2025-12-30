package sqlite

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
		logger.Debug().Msgf("run maintenance script[%d]: %q", idx, sql)

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

		logger.Debug().Msgf("run maintenance script[%d] finished; row affected: %d", idx, rowsaffected)
	}

	// print some stats
	var numEpisodes, numPodcasts int
	if err := dbi.GetContext(ctx, &numEpisodes, "SELECT count(*) FROM episodes"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count episodes failed")
	}

	if err := dbi.GetContext(ctx, &numPodcasts, "SELECT count(*) FROM podcasts"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count podcasts failed")
	}

	logger.Info().Msgf("database maintenance finished; podcasts: %d; episodes: %d", numPodcasts, numEpisodes)

	return nil
}

//------------------------------------------------------------------------------

//nolint:gochecknoglobals
var maintScripts = []string{
	// delete play actions when for given episode never play action exists
	`DELETE FROM episodes AS e
		WHERE action = 'play'
		AND updated_at < datetime('now','-14 day')
		AND EXISTS (
			SELECT NULL FROM episodes AS ed
			WHERE ed.url = e.url AND ed.action = 'play' AND ed.updated_at > e.updated_at);`,
	`VACUUM;`,
	`ANALYZE;`,
	`PRAGMA optimize;`,
}

//------------------------------------------------------------------------------
