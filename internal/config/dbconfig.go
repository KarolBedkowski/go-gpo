package config

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
	if d.Connstr == "" {
		return ConfigurationError("db.connstr argument can't be empty")
	}

	if d.Driver == "" {
		return ConfigurationError("db.driver argument can't be empty")
	} else if d.Driver != "sqlite3" && d.Driver != "postgres" { //nolint:goconst
		return ConfigurationError("invalid (unsupported) db.driver")
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
