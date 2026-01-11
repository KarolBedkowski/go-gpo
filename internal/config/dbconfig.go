package config

//
// dbconfig.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

type DBConfig struct {
	Driver  string
	Connstr string
}

func NewDBConfig(driver, connstr string) DBConfig {
	return DBConfig{
		Driver:  mapDriverName(driver),
		Connstr: connstr,
	}
}

func (d *DBConfig) Validate() error {
	if d.Connstr == "" {
		return aerr.New("db.connstr argument can't be empty").WithTag(aerr.ValidationError)
	}

	if d.Driver == "" {
		return aerr.New("db.driver argument can't be empty").WithTag(aerr.ValidationError)
	} else if d.Driver != "sqlite3" && d.Driver != "postgres" { //nolint:goconst
		return aerr.New("invalid (unsupported) db.driver").WithTag(aerr.ValidationError)
	}

	return nil
}

func mapDriverName(driver string) string {
	switch driver {
	case "sqlite", "sqlite3":
		return "sqlite3"
	case "pg", "postgresql", "postgres":
		return "postgres"
	}

	return driver
}
