package web

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type podcastPages struct {
	podcastsSrv      *service.PodcastsSrv
	subscriptionsSrv *service.SubscriptionsSrv
	template         templates
	webroot          string
}

func newPodcastPages(i do.Injector) (podcastPages, error) {
	return podcastPages{
		podcastsSrv:      do.MustInvoke[*service.PodcastsSrv](i),
		subscriptionsSrv: do.MustInvoke[*service.SubscriptionsSrv](i),
		template:         do.MustInvoke[templates](i),
		webroot:          do.MustInvokeNamed[string](i, "server.webroot"),
	}, nil
}

func (p podcastPages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, srvsupport.Wrap(p.list))
	r.Post(`/`, srvsupport.Wrap(p.addPodcast))
	r.Get(`/{podcastid:[0-9]+}/`, srvsupport.Wrap(p.podcastGet))
	r.Post(`/{podcastid:[0-9]+}/unsubscribe`, srvsupport.Wrap(p.podcastUnsubscribe))

	return r
}

func (p podcastPages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	podcasts, err := p.podcastsSrv.GetPodcastsWithLastEpisode(ctx, user)
	if err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get user podcasts error")

		return
	}

	data := struct {
		Podcasts []model.PodcastWithLastEpisode
	}{
		Podcasts: podcasts,
	}

	if err := p.template.executeTemplate(w, "podcasts.tmpl", &data); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		srvsupport.WriteError(w, r, http.StatusInternalServerError, "")
	}
}

func (p podcastPages) addPodcast(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	if err := r.ParseForm(); err != nil {
		logger.Error().Err(err).Msg("parse form error")
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")
	}

	var podcast string
	if podcasts, ok := r.Form["url"]; ok && len(podcasts) == 1 {
		podcast = strings.TrimSpace(podcasts[0])
	}

	if podcast == "" {
		p.list(ctx, w, r, logger)

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   internal.ContextUser(ctx),
		DeviceName: "",
		Add:        []string{podcast},
		Timestamp:  time.Now(),
	}

	if _, err := p.subscriptionsSrv.ChangeSubscriptions(ctx, &cmd); err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("add podcast error")

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}

func (p podcastPages) podcastGet(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	podcastid, ok := podcastFromURLParam(r)
	if !ok {
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	user := internal.ContextUser(ctx)

	podcast, err := p.podcastsSrv.GetPodcast(ctx, user, podcastid)
	if errors.Is(err, repository.ErrNoData) {
		srvsupport.WriteError(w, r, http.StatusNotFound, "")

		return
	} else if err != nil {
		logger.Error().Err(err).Int64("podcast_id", podcastid).Msg("get podcast failed")
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	data := struct {
		Podcast model.Podcast
	}{Podcast: podcast}

	if err := p.template.executeTemplate(w, "podcast.tmpl", &data); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		srvsupport.WriteError(w, r, http.StatusInternalServerError, "")
	}
}

func (p podcastPages) podcastUnsubscribe(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	podcastid, ok := podcastFromURLParam(r)
	if !ok {
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	user := internal.ContextUser(ctx)

	podcast, err := p.podcastsSrv.GetPodcast(ctx, user, podcastid)
	if errors.Is(err, repository.ErrNoData) {
		srvsupport.WriteError(w, r, http.StatusNotFound, "")

		return
	} else if err != nil {
		logger.Error().Err(err).Int64("podcast_id", podcastid).Msg("get podcast failed")
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   user,
		DeviceName: "",
		Remove:     []string{podcast.URL},
		Timestamp:  time.Now(),
	}

	if _, err := p.subscriptionsSrv.ChangeSubscriptions(ctx, &cmd); err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("add podcast error")

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}

func podcastFromURLParam(r *http.Request) (int64, bool) {
	podcastidS := chi.URLParam(r, "podcastid")
	if podcastidS == "" {
		return 0, false
	}

	podcastid, err := strconv.ParseInt(podcastidS, 10, 64)
	if err != nil {
		return 0, false
	}

	return podcastid, true
}
