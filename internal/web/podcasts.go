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
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type podcastPages struct {
	podcastsSrv      *service.PodcastsSrv
	subscriptionsSrv *service.SubscriptionsSrv
	webroot          string
	renderer         *nt.Renderer
}

func newPodcastPages(i do.Injector) (podcastPages, error) {
	return podcastPages{
		podcastsSrv:      do.MustInvoke[*service.PodcastsSrv](i),
		subscriptionsSrv: do.MustInvoke[*service.SubscriptionsSrv](i),
		webroot:          do.MustInvokeNamed[string](i, "server.webroot"),
		renderer:         do.MustInvoke[*nt.Renderer](i),
	}, nil
}

func (p podcastPages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, srvsupport.Wrap(p.list))
	r.Post(`/`, srvsupport.Wrap(p.addPodcast))
	r.Get(`/{podcastid:[0-9]+}/`, srvsupport.Wrap(p.podcastGet))
	r.Post(`/{podcastid:[0-9]+}/unsubscribe`, srvsupport.Wrap(p.podcastUnsubscribe))
	r.Post(`/{podcastid:[0-9]+}/resubscribe`, srvsupport.Wrap(p.podcastResubscribe))
	r.Get(`/{podcastid:[0-9]+}/delete`, srvsupport.Wrap(p.podcastDeleteGet))
	r.Post(`/{podcastid:[0-9]+}/delete`, srvsupport.Wrap(p.podcastDeletePost))

	return r
}

func (p podcastPages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)

	subscribedOnly := !r.URL.Query().Has("showall")

	logger.Debug().Interface("showall", r.URL.Query().Get("showall")).Msg("args")

	podcasts, err := p.podcastsSrv.GetPodcastsWithLastEpisode(ctx, user, subscribedOnly)
	if err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get user podcasts error")

		return
	}

	p.renderer.WritePage(w, &nt.PodcastsPage{Podcasts: podcasts, SubscribedOnly: subscribedOnly})
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
		UserName:   common.ContextUser(ctx),
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
	podcast, status := p.podcastFromURLParam(ctx, r, logger)
	if status > 0 {
		srvsupport.WriteError(w, r, status, "")

		return
	}

	p.renderer.WritePage(w, &nt.PodcastPage{Podcast: podcast})
}

func (p podcastPages) podcastUnsubscribe(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	podcast, status := p.podcastFromURLParam(ctx, r, logger)
	if status > 0 || podcast == nil {
		srvsupport.WriteError(w, r, status, "")

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   common.ContextUser(ctx),
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

func (p podcastPages) podcastResubscribe(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	podcast, status := p.podcastFromURLParam(ctx, r, logger)
	if status > 0 || podcast == nil {
		srvsupport.WriteError(w, r, status, "")

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   common.ContextUser(ctx),
		DeviceName: "",
		Add:        []string{podcast.URL},
		Timestamp:  time.Now(),
	}

	if _, err := p.subscriptionsSrv.ChangeSubscriptions(ctx, &cmd); err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("add podcast error")

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}

func (p podcastPages) podcastFromURLParam(ctx context.Context, r *http.Request, logger *zerolog.Logger,
) (*model.Podcast, int) {
	podcastidS := chi.URLParam(r, "podcastid")
	if podcastidS == "" {
		return nil, http.StatusBadRequest
	}

	podcastid, err := strconv.ParseInt(podcastidS, 10, 32)
	if err != nil {
		return nil, http.StatusBadRequest
	}

	user := common.ContextUser(ctx)

	podcast, err := p.podcastsSrv.GetPodcast(ctx, user, podcastid)
	if errors.Is(err, common.ErrNoData) {
		return nil, http.StatusNotFound
	} else if err != nil {
		logger.Error().Err(err).Int64("podcast_id", podcastid).Msg("get podcast failed")

		return nil, http.StatusNotFound
	}

	return podcast, 0
}

func (p podcastPages) podcastDeleteGet(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	podcast, status := p.podcastFromURLParam(ctx, r, logger)
	if status > 0 || podcast == nil {
		srvsupport.WriteError(w, r, status, "")

		return
	}

	p.renderer.WritePage(w, &nt.PodcastDeletePage{Podcast: podcast})
}

func (p podcastPages) podcastDeletePost(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	podcastidS := chi.URLParam(r, "podcastid")
	if podcastidS == "" {
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	podcastid, err := strconv.ParseInt(podcastidS, 10, 32)
	if err != nil || podcastid < 1 {
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	user := common.ContextUser(ctx)

	if err := p.podcastsSrv.DeletePodcast(ctx, user, podcastid); err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("delete podcast error")

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}
