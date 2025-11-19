// simple.go
// /subscriptions/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/opml"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type simpleResource struct {
	subServ *service.SubscriptionsSrv
}

func newSimpleResource(i do.Injector) (simpleResource, error) {
	return simpleResource{
		subServ: do.MustInvoke[*service.SubscriptionsSrv](i),
	}, nil
}

func (s *simpleResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.{format}`, srvsupport.Wrap(s.downloadUserSubscriptions))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.{format}`, srvsupport.Wrap(s.downloadDevSubscriptions))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Put(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.{format}`, srvsupport.Wrap(s.uploadSubscriptions))

	return r
}

func (s *simpleResource) downloadUserSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)

	subs, err := s.subServ.GetUserSubscriptions(ctx, &query.GetUserSubscriptionsQuery{UserName: user})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get user subscriptions error")

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml": //nolint:goconst
		o := opml.NewOPML("go-gpo")
		o.AddURL(subs...)

		result, err := o.XML()
		if err != nil {
			logger.Warn().Err(err).Msg("get opml xml error")
			writeError(w, r, http.StatusBadRequest)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(result)
	case "json": //nolint:goconst
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, subs)
	case "txt": //nolint:goconst
		w.WriteHeader(http.StatusOK)
		render.PlainText(w, r, strings.Join(subs, "\n"))
	default:
		logger.Info().Msgf("unknown format %q", format)
		writeError(w, r, http.StatusNotFound)
	}
}

func (s *simpleResource) downloadDevSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	devicename := internal.ContextDevice(ctx)

	subs, err := s.subServ.GetSubscriptions(ctx, &query.GetSubscriptionsQuery{UserName: user, DeviceName: devicename})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get device subscriptions error")

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		result, err := formatOMPL(subs)
		if err != nil {
			logger.Warn().Err(err).Msg("build opml error")
			writeError(w, r, http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(result)
	case "json":
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, subs)
	case "txt":
		w.WriteHeader(http.StatusOK)
		render.PlainText(w, r, strings.Join(subs, "\n"))
	default:
		logger.Info().Msgf("unknown format %q", format)
		writeError(w, r, http.StatusNotFound)
	}
}

func (s *simpleResource) uploadSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	devicename := internal.ContextDevice(ctx)

	var (
		subs []string
		err  error
	)

	format := chi.URLParam(r, "format")
	switch format {
	case "opml":
		subs, err = parseOPML(r.Body)
	case "json":
		err = render.DecodeJSON(r.Body, &subs)
	case "txt":
		subs, err = parseTextSubs(r.Body)
	default:
		logger.Debug().Msgf("unknown format %q", format)
		writeError(w, r, http.StatusNotFound)

		return
	}

	if err != nil {
		logger.Debug().Err(err).Msgf("parse %q error", format)
		writeError(w, r, http.StatusBadRequest)

		return
	}

	cmd := command.ReplaceSubscriptionsCmd{
		UserName:      user,
		DeviceName:    devicename,
		Subscriptions: subs,
		Timestamp:     time.Now().UTC(),
	}
	if err := s.subServ.ReplaceSubscriptions(ctx, &cmd); err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("update subscriptions error")
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func formatOMPL(subs []string) ([]byte, error) {
	o := opml.NewOPML("go-gpo")
	o.AddURL(subs...)

	result, err := o.XML()
	if err != nil {
		return nil, fmt.Errorf("build opml error: %w", err)
	}

	return result, nil
}
