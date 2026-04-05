package config

import "errors"

//
// dbconfig.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

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
	var errs error

	if d.Connstr == "" {
		errs = errors.Join(errs, ConfigurationError("db.connstr argument can't be empty"))
	}

	if d.Driver == "" {
		errs = errors.Join(errs, ConfigurationError("db.driver argument can't be empty"))
	} else if d.Driver != "sqlite3" && d.Driver != "postgres" { //nolint:goconst
		errs = errors.Join(errs, newConfigurationError("invalid (unsupported) db.driver %q", d.Driver))
	}

	return errs
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
