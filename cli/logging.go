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
	"github.com/rs/zerolog/log"
)

// InitializeLogger set log level and optional log filename.
func initializeLogger(level, format string) {
	var llog zerolog.Logger

	switch format {
	default:
		log.Error().Msgf("logger: unknown log format %q; using logfmt", format)

		fallthrough
	case "logfmt":
		llog = log.Output(zerolog.ConsoleWriter{ //nolint:exhaustruct
			Out:        os.Stderr,
			NoColor:    !outputIsConsole(),
			TimeFormat: time.RFC3339,
		})
	case "json":
		llog = log.Logger
	}

	if l, err := zerolog.ParseLevel(level); err == nil {
		zerolog.SetGlobalLevel(l)
	} else {
		log.Error().Msgf("logger: unknown log level %q; using debug", level)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = llog.With().Caller().Logger()

	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

func outputIsConsole() bool {
	fileInfo, _ := os.Stdout.Stat()

	return fileInfo != nil && (fileInfo.Mode()&os.ModeCharDevice) != 0
}
