// simple.go
// /subscriptions/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/oxtyped/go-opml/opml"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/repository"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

type simpleResource struct {
	cfg     *Configuration
	repo    *repository.Repository
	subServ *service.Subs
}

func (s *simpleResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !s.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
		r.Use(checkUserMiddleware)
	}

	r.Get("/{user:[0-9a-z._-]+}.{format}", s.downloadAllSubscriptions)
	r.Get("/{user:[0-9a-z._-]+}/{deviceid:[0-9a-z._-]+}.{format}", s.downloadSubscriptions)
	r.Put("/{user:[0-9a-z._-]+}/{deviceid:[0-9a-z._-]+}.{format}", s.uploadSubscriptions)

	return r
}

func (s *simpleResource) downloadAllSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	subs, err := s.subServ.GetUserSubscriptions(ctx, user, time.Time{})
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		w.WriteHeader(http.StatusBadRequest)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		o := opml.NewOPMLFromBlank("go-gpodder")
		for _, s := range subs {
			o.AddRSSFromURL(s, 2*time.Second)
		}

		result, err := o.XML()
		if err != nil {
			logger.Info().Err(err).Msg("get opml xml error")
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	case "json":
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, subs)
	case "txt":
		w.WriteHeader(http.StatusOK)
		render.PlainText(w, r, strings.Join(subs, "\n"))
	default:
		logger.Info().Msgf("unknown format %q", format)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *simpleResource) downloadSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	subs, err := s.subServ.GetDeviceSubscriptions(ctx, user, deviceid, time.Time{})
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		w.WriteHeader(http.StatusBadRequest)

		return
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Info().Msgf("unknown device: %q", deviceid)
		w.WriteHeader(http.StatusBadRequest)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		o := opml.NewOPMLFromBlank("go-gpodder")
		for _, s := range subs {
			o.AddRSSFromURL(s, 2*time.Second)
		}

		result, err := o.XML()
		if err != nil {
			logger.Info().Err(err).Msg("get opml xml error")
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	case "json":
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, subs)
	case "txt":
		w.WriteHeader(http.StatusOK)
		render.PlainText(w, r, strings.Join(subs, "\n"))
	default:
		logger.Info().Msgf("unknown format %q", format)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *simpleResource) uploadSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var subs []string

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		// TODO need tests
		var buf bytes.Buffer

		count, err := io.Copy(&buf, r.Body)
		if err != nil {
			logger.Warn().Err(err).Msgf("parse opml - copy body - error")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if count == 0 {
			logger.Debug().Msgf("parse opml error - empty body")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		o, err := opml.NewOPML(buf.Bytes())
		if err != nil {
			logger.Warn().Err(err).Msgf("parse opml error")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		for _, i := range o.Outlines() {
			if url := i.XMLURL; url != "" {
				subs = append(subs, url)
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
		return

	case "json":
		if err := render.DecodeJSON(r.Body, &subs); err != nil {
			logger.Warn().Err(err).Msgf("parse json error")
			w.WriteHeader(http.StatusBadRequest)

			return
		}
	case "txt":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Warn().Err(err).Msgf("read body error")
			w.WriteHeader(http.StatusBadRequest)
		}

		subs = slices.Collect(strings.Lines(string(body)))
	default:
		logger.Info().Msgf("unknown format %q", format)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.subServ.UpdateDeviceSubscriptions(ctx, user, deviceid, subs, time.Now()); err != nil {
		logger.Debug().Strs("subs", subs).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")

		w.WriteHeader(http.StatusBadRequest)

		return
	}

	w.WriteHeader(http.StatusOK)
}
