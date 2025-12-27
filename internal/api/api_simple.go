package api

// simple.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/formats"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
)

// simpleResource handle request to simple api (/subscriptions/ resource).
type simpleResource struct {
	subServ *service.SubscriptionsSrv
}

func newSimpleResource(i do.Injector) (simpleResource, error) {
	return simpleResource{
		subServ: do.MustInvoke[*service.SubscriptionsSrv](i),
	}, nil
}

func (s *simpleResource) Routes() *chi.Mux {
	router := chi.NewRouter()

	// base: /subscriptions/

	router.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.{format}`, srvsupport.WrapNamed(s.downloadUserSubscriptions, "api_subs_user"))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.{format}`,
			srvsupport.WrapNamed(s.downloadDevSubscriptions, "api_subs_userdev"))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Put(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.{format}`,
			srvsupport.WrapNamed(s.uploadSubscriptions, "api_subs_userdev_put"))

	return router
}

func (s *simpleResource) downloadUserSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)

	subs, err := s.subServ.GetUserSubscriptions(ctx, &query.GetUserSubscriptionsQuery{UserName: user})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get user subscriptions error")

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml": //nolint:goconst
		o := formats.NewOPML("go-gpo")
		for _, s := range subs {
			o.AddRSS(s.URL, s.Title, s.Title)
		}

		w.WriteHeader(http.StatusOK)
		render.XML(w, r, &o)
	case "json": //nolint:goconst
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, subs.ToURLs())
	case "jsonp": //nolint:goconst
		w.WriteHeader(http.StatusOK)
		render.JSON(newJSONPWriter(r, w), r, subs.ToURLs())
	case "txt": //nolint:goconst
		w.WriteHeader(http.StatusOK)
		render.PlainText(w, r, strings.Join(subs.ToURLs(), "\n"))
	case "xml":
		xmlsubs := formats.NewXMLPodcasts(subs)

		w.WriteHeader(http.StatusOK)
		render.XML(w, r, &xmlsubs)
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
	user := common.ContextUser(ctx)
	devicename := common.ContextDevice(ctx)

	subs, err := s.subServ.GetSubscriptions(ctx, &query.GetSubscriptionsQuery{UserName: user, DeviceName: devicename})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get device subscriptions error")

		return
	}

	switch format := chi.URLParam(r, "format"); format {
	case "opml":
		o := formats.NewOPML("go-gpo")
		for _, s := range subs {
			o.AddRSS(s.URL, s.Title, s.Title)
		}

		w.WriteHeader(http.StatusOK)
		render.XML(w, r, &o)
	case "json":
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, subs.ToURLs())
	case "jsonp":
		w.WriteHeader(http.StatusOK)
		render.JSON(newJSONPWriter(r, w), r, subs.ToURLs())
	case "xml":
		xmlsubs := formats.NewXMLPodcasts(subs)

		w.WriteHeader(http.StatusOK)
		render.XML(w, r, &xmlsubs)
	case "txt":
		w.WriteHeader(http.StatusOK)
		render.PlainText(w, r, strings.Join(subs.ToURLs(), "\n"))
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
	user := common.ContextUser(ctx)
	devicename := common.ContextDevice(ctx)

	var (
		subs []string
		err  error
	)

	format := chi.URLParam(r, "format")
	switch format {
	case "opml":
		subs, err = parseOPML(r.Body)
	case "json", "jsonp":
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

// ------------------------------------------------------
