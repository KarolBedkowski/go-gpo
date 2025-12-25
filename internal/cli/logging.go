// logging.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package cli

import (
	"fmt"
	"io"
	stdlog "log"
	"log/syslog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/journald"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// InitializeLogger set log level and optional log filename.
func initializeLogger(level, format string) error { //nolint:cyclop
	zerolog.ErrorMarshalFunc = aerr.ErrorMarshalFunc //nolint:reassign

	var writer io.Writer

	switch format {
	case "json":
		writer = os.Stderr

	case "text":
		writer = zerolog.ConsoleWriter{ //nolint:exhaustruct
			Out:          os.Stderr,
			NoColor:      true,
			PartsExclude: []string{zerolog.TimestampFieldName},
		}

	case "syslog":
		syslogwriter, err := syslog.New(syslog.LOG_USER, "gogpo")
		if err != nil {
			return fmt.Errorf("init syslog error: %w", err)
		}

		writer = zerolog.SyslogLevelWriter(syslogwriter)

	case "journald":
		writer = journald.NewJournalDWriter()

	case "logfmt":
		writer = setupLogfmtConsoleWriter()

	default:
		if format != "" && format != "console" {
			log.Error().Msgf("logger: unknown log format %q; using default", format)
		}

		writer = setupConsoleWriter()
	}

	log.Logger = log.Output(writer).With().Timestamp().Caller().Logger()

	if l, err := zerolog.ParseLevel(level); err == nil {
		zerolog.SetGlobalLevel(l)
	} else {
		log.Error().Msgf("logger: unknown log level %q; using debug", level)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	return nil
}

func setupConsoleWriter() io.Writer {
	console := outputIsConsole()

	// log full datetime when log is written to file; skip date on console.
	tformat := time.RFC3339
	if console {
		tformat = time.TimeOnly
	}

	return zerolog.ConsoleWriter{ //nolint:exhaustruct
		Out:        os.Stderr,
		NoColor:    !console,
		TimeFormat: tformat,
	}
}

func outputIsConsole() bool {
	fileInfo, _ := os.Stderr.Stat()

	return fileInfo != nil && (fileInfo.Mode()&os.ModeCharDevice) != 0
}

// setupLogfmtConsoleWriter configure logger to proper logfmt format (all fields are in form key=val).
func setupLogfmtConsoleWriter() io.Writer {
	return zerolog.ConsoleWriter{ //nolint:exhaustruct
		Out:        os.Stderr,
		NoColor:    true,
		TimeFormat: time.RFC3339,
		FormatLevel: func(i any) string {
			if i == nil {
				return ""
			} else {
				return fmt.Sprintf("level=%s", i)
			}
		},
		FormatTimestamp: func(i any) string { return fmt.Sprintf("ts=%s", i) },
		FormatMessage: func(i any) string {
			if i == nil {
				return "msg=<nil>"
			} else {
				return "msg=" + strconv.Quote(fmt.Sprintf("%s", i))
			}
		},
		FormatCaller: func(i any) string {
			if i == nil {
				return "UNKNOWN"
			} else {
				c := fmt.Sprintf("%s", i)
				if strings.ContainsAny(c, " \"") {
					c = strconv.Quote(c)
				}

				return "caller=" + c
			}
		},
	}
}
