// simple.go
// /subscriptions/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpodder/internal"
	"gitlab.com/kabes/go-gpodder/internal/opml"
	"gitlab.com/kabes/go-gpodder/internal/repository"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

type simpleResource struct {
	cfg     *Configuration
	repo    *repository.Database
	subServ *service.Subs
}

func (s *simpleResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !s.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
	}

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.{format}`, wrap(s.downloadAllSubscriptions))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.{format}`, wrap(s.downloadSubscriptions))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Put(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.{format}`, wrap(s.uploadSubscriptions))

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
		if errors.Is(err, service.ErrUnknownUser) {
			logger.Info().Msgf("unknown user: %q", user)
			writeError(w, r, http.StatusBadRequest, nil)
		} else {
			logger.Info().Err(err).Msg("update device error")
			writeError(w, r, http.StatusInternalServerError, nil)
		}

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml": //nolint:goconst
		o := opml.NewOPML("go-gpodder")
		o.AddURL(subs...)

		result, err := o.XML()
		if err != nil {
			logger.Info().Err(err).Msg("get opml xml error")
			writeError(w, r, http.StatusInternalServerError, nil)

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
		writeError(w, r, http.StatusBadRequest, nil)
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
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Info().Msgf("unknown device: %q", deviceid)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		result, err := formatOMPL(subs)
		if err != nil {
			logger.Info().Err(err).Msg("build opml error")
			writeError(w, r, http.StatusInternalServerError, nil)

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
		writeError(w, r, http.StatusBadRequest, nil)
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
		logger.Info().Msgf("unknown format %q", format)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err != nil {
		logger.Warn().Err(err).Msgf("parse %q error", format)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err := s.subServ.UpdateDeviceSubscriptions(ctx, user, deviceid, subs, time.Now()); err != nil {
		logger.Debug().Strs("subs", subs).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")
		writeError(w, r, http.StatusInternalServerError, nil)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func formatOMPL(subs []string) ([]byte, error) {
	o := opml.NewOPML("go-gpodder")
	o.AddURL(subs...)

	result, err := o.XML()
	if err != nil {
		return nil, fmt.Errorf("build opml error: %w", err)
	}

	return result, nil
}
