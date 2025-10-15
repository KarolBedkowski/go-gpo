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

var showVersion = flag.Bool("version", false, "Print version information.")

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("go-gpodder version %s\n", Version)
		os.Exit(0)
	}

	initializeLogger("debug", "logfmt")

	log.Logger.Log().Msg("Starting...")

	re := &repository.Repository{}
	re.Connect("sqlite3", "database.sqlite")

	api.Start(re)
}
