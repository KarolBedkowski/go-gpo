// simple.go
// /subscriptions/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/opml"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type simpleResource struct {
	subServ *service.Subs
}

func newSimpleResource(i do.Injector) (simpleResource, error) {
	return simpleResource{
		subServ: do.MustInvoke[*service.Subs](i),
	}, nil
}

func (s *simpleResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.{format}`, internal.Wrap(s.downloadAllSubscriptions))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.{format}`, internal.Wrap(s.downloadSubscriptions))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Put(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.{format}`, internal.Wrap(s.uploadSubscriptions))

	return r
}

func (s *simpleResource) downloadAllSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)

	subs, err := s.subServ.GetUserSubscriptions(ctx, user, time.Time{})
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get user subscriptions error")

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml": //nolint:goconst
		o := opml.NewOPML("go-gpo")
		o.AddURL(subs...)

		result, err := o.XML()
		if err != nil {
			logger.Warn().Err(err).Str("mod", "api").Msg("get opml xml error")
			internal.WriteError(w, r, http.StatusBadRequest, "invalid opml content")

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
		internal.WriteError(w, r, http.StatusNotFound, "")
	}
}

func (s *simpleResource) downloadSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	subs, err := s.subServ.GetDeviceSubscriptions(ctx, user, deviceid, time.Time{})
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get device subscriptions error")

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		result, err := formatOMPL(subs)
		if err != nil {
			logger.Warn().Err(err).Str("mod", "api").Msg("build opml error")
			internal.WriteError(w, r, http.StatusInternalServerError, "")

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
		internal.WriteError(w, r, http.StatusNotFound, "")
	}
}

func (s *simpleResource) uploadSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

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
		var body []byte

		body, err = io.ReadAll(r.Body)
		if err == nil {
			subs = slices.Collect(strings.Lines(string(body)))
		}
	default:
		logger.Debug().Msgf("unknown format %q", format)
		internal.WriteError(w, r, http.StatusNotFound, "")

		return
	}

	if err != nil {
		logger.Debug().Err(err).Msgf("parse %q error", format)
		internal.WriteError(w, r, http.StatusBadRequest, "invalid request data")

		return
	}

	subscribed := model.NewSubscribedURLS(subs)
	if err := s.subServ.UpdateDeviceSubscriptions(ctx, user, deviceid, subscribed, time.Now()); err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("update subscriptions error")
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
