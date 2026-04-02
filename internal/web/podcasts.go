package web

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"net/http"
	"strconv"
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
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
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
	r.Get(`/`, srvsupport.WrapNamed(p.list, "web_podcast_index"))
	r.Get(`/export`, srvsupport.WrapNamed(p.export, "web_podcast_export"))
	r.Post(`/`, srvsupport.WrapNamed(p.addPodcast, "web_podcast_add"))
	r.Get(`/{podcastid:[0-9]+}/`, srvsupport.WrapNamed(p.podcastGet, "web_podcast_get"))
	r.Post(`/{podcastid:[0-9]+}/unsubscribe`, srvsupport.WrapNamed(p.podcastUnsubscribe, "web_podcast_unsub"))
	r.Post(`/{podcastid:[0-9]+}/resubscribe`, srvsupport.WrapNamed(p.podcastResubscribe, "web_podcast_resub"))
	r.Get(`/{podcastid:[0-9]+}/delete`, srvsupport.WrapNamed(p.podcastDeleteGet, "web_podcast_del"))
	r.Post(`/{podcastid:[0-9]+}/delete`, srvsupport.WrapNamed(p.podcastDeletePost, "web_podcast_del_post"))

	return r
}

func (p podcastPages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)
	subscribedOnly := !r.URL.Query().Has("showall")

	podcasts, err := p.podcastsSrv.GetPodcastsWithLastEpisode(ctx, user, subscribedOnly)
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: get user_name=%s podcasts error=%q", user, err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	p.renderer.WritePage(w, &nt.PodcastsPage{Podcasts: podcasts, SubscribedOnly: subscribedOnly})
}

func (p podcastPages) addPodcast(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	const maxBody = 1024 * 1024 // 1k

	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseForm(); err != nil {
		logger.Error().Err(err).Msgf("web.Podcasts: bad request - parse form error=%q", err)
		p.renderer.WriteError(ctx, w, err)
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

	_ = cmd.Sanitize()
	if len(cmd.Add) != 1 {
		p.renderer.WriteBadRequestError(ctx, w, "invalid podcast URL")
	}

	if _, err := p.subscriptionsSrv.ChangeSubscriptions(ctx, &cmd); err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: add podcast error=%q", err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}

func (p podcastPages) podcastGet(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	podcast, err := p.podcastFromURLParam(ctx, r, logger)
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: get podcast error=%q", err)
		p.renderer.WriteError(ctx, w, err)

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
	podcast, err := p.podcastFromURLParam(ctx, r, logger)
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: get podcast error=%q", err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   common.ContextUser(ctx),
		DeviceName: "",
		Remove:     []string{podcast.URL},
		Timestamp:  time.Now(),
	}

	if _, err := p.subscriptionsSrv.ChangeSubscriptions(ctx, &cmd); err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msgf("web.Podcasts: add podcast error=%q", err)
		p.renderer.WriteError(ctx, w, err)

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
	podcast, err := p.podcastFromURLParam(ctx, r, logger)
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: get podcast error=%q", err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   common.ContextUser(ctx),
		DeviceName: "",
		Add:        []string{podcast.URL},
		Timestamp:  time.Now(),
	}

	if _, err := p.subscriptionsSrv.ChangeSubscriptions(ctx, &cmd); err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: resubscribe podcast_url=%q error=%q", podcast.URL, err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}

func (p podcastPages) podcastFromURLParam(ctx context.Context, r *http.Request, logger *zerolog.Logger,
) (*model.Podcast, error) {
	podcastidS := chi.URLParam(r, "podcastid")
	if podcastidS == "" {
		return nil, aerr.ErrBadRequest.WithMsg("invalid podcast id")
	}

	podcastid, err := strconv.ParseInt(podcastidS, 10, 32)
	if err != nil {
		return nil, aerr.ErrBadRequest.WithMsg("invalid podcast id")
	}

	user := common.ContextUser(ctx)

	podcast, err := p.podcastsSrv.GetPodcast(ctx, user, podcastid)
	if err != nil {
		return nil, aerr.Wrap(err)
	}

	return podcast, nil
}

func (p podcastPages) podcastDeleteGet(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	podcast, err := p.podcastFromURLParam(ctx, r, logger)
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: get podcast error=%q", err)
		p.renderer.WriteError(ctx, w, err)

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
		p.renderer.WriteBadRequestError(ctx, w, "Invalid podcast id.")

		return
	}

	podcastid, err := strconv.ParseInt(podcastidS, 10, 32)
	if err != nil || podcastid < 1 {
		p.renderer.WriteBadRequestError(ctx, w, "Invalid podcast id.")

		return
	}

	user := common.ContextUser(ctx)

	if err := p.podcastsSrv.DeletePodcast(ctx, user, podcastid); err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: delete podcast_id=%d error=%q", podcastid, err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	http.Redirect(w, r, p.webroot+"/web/podcast/", http.StatusFound)
}

func (p podcastPages) export(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)

	subs, err := p.subscriptionsSrv.GetUserSubscriptions(ctx, &query.GetUserSubscriptionsQuery{UserName: user})
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Podcasts: get subscribed podcasts for export user_name=%s error=%q", user, err)
		p.renderer.WriteError(ctx, w, err)

		return
	}

	o := formats.NewOPML("go-gpo")
	for _, s := range subs {
		o.AddRSS(s.URL, s.Title, s.Title)
	}

	w.Header().Add("Content-Disposition", "attachment; filename=\"export.opml\"")
	render.XML(w, r, &o)
}
