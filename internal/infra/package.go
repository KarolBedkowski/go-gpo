// Package infra provide infrastructure layer.
package infra

//
// package.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"database/sql"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/infra/pg"
	"gitlab.com/kabes/go-gpo/internal/infra/sqlite"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var ErrInvalidDBInfra = aerr.New("not found infrastructure for db driver")

//nolint:gochecknoglobals
var Package = do.Package(
	do.Lazy(func(i do.Injector) (repository.Sessions, error) {
		switch getDriverName(i) {
		case "sqlite3": //nolint:goconst
			return &sqlite.Repository{}, nil
		case "postgres": //nolint:goconst
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),
	do.Lazy(func(i do.Injector) (repository.Users, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return &sqlite.Repository{}, nil
		case "postgres":
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),
	do.Lazy(func(i do.Injector) (repository.Podcasts, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return &sqlite.Repository{}, nil
		case "postgres":
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),
	do.Lazy(func(i do.Injector) (repository.Devices, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return &sqlite.Repository{}, nil
		case "postgres":
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),
	do.Lazy(func(i do.Injector) (repository.Episodes, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return &sqlite.Repository{}, nil
		case "postgres":
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),
	do.Lazy(func(i do.Injector) (repository.Settings, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return &sqlite.Repository{}, nil
		case "postgres":
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),
	do.Lazy(func(i do.Injector) (repository.Maintenance, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return &sqlite.Repository{}, nil
		case "postgres":
			return &pg.Repository{}, nil
		default:
			return nil, ErrInvalidDBInfra
		}
	}),

	do.Lazy(func(i do.Injector) (repository.Database, error) {
		switch getDriverName(i) {
		case "sqlite3":
			return sqlite.NewDatabaseI(i)
		case "postgres":
			return pg.NewDatabaseI(i)
		default:
			return nil, ErrInvalidDBInfra
		}
	}),

	do.Transient(func(i do.Injector) (*sql.DB, error) {
		dbimpl, err := do.Invoke[repository.Database](i)
		if err != nil {
			return nil, aerr.Wrapf(err, "get database failed")
		}

		if db := dbimpl.GetDB(); db != nil {
			return db, nil
		}

		return nil, aerr.Wrapf(err, "get database connection failed")
	}),
)

func getDriverName(i do.Injector) string {
	dbconf := do.MustInvoke[config.DBConfig](i)

	return dbconf.Driver
}
