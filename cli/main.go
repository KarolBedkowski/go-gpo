package main

//
// prom-logmonitor.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	// _ "github.com/WAY29/icecream-go/icecream".

	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"gitlab.com/kabes/go-gpodder/internal/api"
	"gitlab.com/kabes/go-gpodder/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

var (
	Version   = ""
	Revision  = ""
	BuildDate = ""
	BuildUser = ""
	Branch    = ""
)

var (
	showVersion = flag.Bool("version", false, "Print version information.")
	noauth      = flag.Bool("no-auth", false, "Disable authentication.")
	database    = flag.String("database", "database.sqlite", "Database file.")
	loglevel    = flag.String("log.level", "info", "Log level (debug, info, warn, error).")
	logformat   = flag.String("log.format", "logfmt", "Log format (logfmt, json).")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("go-gpodder version %s\n", Version)
		os.Exit(0)
	}

	initializeLogger(*loglevel, *logformat)

	log.Logger.Log().Msg("Starting...")

	re := &repository.Repository{}
	re.Connect("sqlite3", (*database)+"?_fk=true")

	cfg := api.Configuration{
		NoAuth: *noauth,
	}

	api.Start(re, &cfg)
}
