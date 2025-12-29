package sqlite

//
// sqlite_maint.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
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

func (Repository) Migrate(ctx context.Context, db *sql.DB) error {
	logger := log.Ctx(ctx)

	migdir, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		panic(fmt.Errorf("prepare migration fs failed: %w", err))
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migdir)
	if err != nil {
		panic(fmt.Errorf("create goose provider failed: %w", err))
	}

	ver, err := provider.GetDBVersion(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "", "failed to check current database version")
	}

	logger.Info().Msgf("current database version: %d", ver)

	for {
		res, err := provider.UpByOne(ctx)
		if res != nil {
			logger.Debug().Msgf("migration: %s", res)
		}

		if errors.Is(err, goose.ErrNoNextVersion) {
			break
		} else if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "", "migrate database up failed")
		}
	}

	ver, err = provider.GetDBVersion(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "", "failed to check current database version")
	}

	logger.Info().Msgf("migrated database version: %d", ver)

	_, err = db.ExecContext(ctx, "PRAGMA optimize")
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute optimize script failed")
	}

	return nil
}

func (Repository) OnOpenConn(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		`PRAGMA temp_store = MEMORY;
		PRAGMA busy_timeout = 1000;
		`,
	)
	if err != nil {
		return aerr.Wrap(err)
	}

	return nil
}

func (Repository) OnCloseConn(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		`PRAGMA optimize`,
	)
	if err != nil {
		return aerr.Wrap(err)
	}

	return nil
}

func (r Repository) Clear(ctx context.Context, db *sql.DB) error {
	sql := `
		PRAGMA foreign_keys=OFF;
		DELETE FROM settings;
		DELETE FROM episodes;
		DELETE FROM podcasts;
		DELETE FROM devices;
		DELETE FROM users;
		DELETE FROM sessions;
		PRAGMA foreign_keys=ON;
	`

	_, err := db.ExecContext(ctx, sql)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "clear database failed")
	}

	return nil
}
