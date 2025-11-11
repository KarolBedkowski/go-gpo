// logging.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package main

import (
	stdlog "log"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/journald"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// InitializeLogger set log level and optional log filename.
func initializeLogger(level, format string) {
	zerolog.ErrorMarshalFunc = aerr.ErrorMarshalFunc //nolint:reassign

	var llog zerolog.Logger

	switch format {
	case "json":
		llog = log.Logger

	case "syslog":
		llog = log.Output(zerolog.ConsoleWriter{ //nolint:exhaustruct
			Out:          os.Stderr,
			NoColor:      true,
			PartsExclude: []string{zerolog.TimestampFieldName},
		})

	case "journald":
		llog = log.Output(journald.NewJournalDWriter())

	default:
		if format != "" && format != "logfmt" {
			log.Error().Msgf("logger: unknown log format %q; using logfmt", format)
		}

		console := outputIsConsole()

		// log full datetime when log is written to file; skip date on console.
		tformat := time.RFC3339
		if console {
			tformat = time.TimeOnly
		}

		llog = log.Output(zerolog.ConsoleWriter{ //nolint:exhaustruct
			Out:        os.Stderr,
			NoColor:    !outputIsConsole(),
			TimeFormat: tformat,
		})
	}

	if l, err := zerolog.ParseLevel(level); err == nil {
		zerolog.SetGlobalLevel(l)
	} else {
		log.Error().Msgf("logger: unknown log level %q; using debug", level)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = llog.With().Timestamp().Caller().Logger()

	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

func outputIsConsole() bool {
	fileInfo, _ := os.Stdout.Stat()

	return fileInfo != nil && (fileInfo.Mode()&os.ModeCharDevice) != 0
}
