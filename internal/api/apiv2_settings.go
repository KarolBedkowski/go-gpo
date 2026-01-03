package api

// updates.g
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

// settingsResource handle /api/2/settings/ request.
type settingsResource struct {
	settingsSrv *service.SettingsSrv
}

func newSettingsResource(i do.Injector) (settingsResource, error) {
	return settingsResource{
		settingsSrv: do.MustInvoke[*service.SettingsSrv](i),
	}, nil
}

func (u settingsResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, srvsupport.WrapNamed(u.getSettings, "api_sett_user"))
	r.With(checkUserMiddleware).
		Post(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, srvsupport.WrapNamed(u.postSettings, "api_sett_user_post"))

	return r
}

func (u settingsResource) getSettings(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)
	key := query.SettingsQuery{
		UserName:   user,
		Scope:      chi.URLParam(r, "scope"),
		DeviceName: r.URL.Query().Get("device"),
		Podcast:    r.URL.Query().Get("podcast"),
		Episode:    r.URL.Query().Get("episode"),
	}

	res, err := u.settingsSrv.GetSettings(ctx, &key)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("SettingsResource: get settings user_name=%s scope=%s error=%q", user, key.Scope, err)

		return
	}

	render.Status(r, http.StatusOK)
	srvsupport.RenderJSON(w, r, &res)
}

func (u settingsResource) postSettings(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)

	var reqData struct {
		Set    map[string]string `json:"set"`
		Remove []string          `json:"remove"`
	}

	if err := render.DecodeJSON(r.Body, &reqData); err != nil {
		logger.Debug().Err(err).Msgf("SettingsResource: decode request from user_name=%s error=%q", user, err)
		writeError(w, r, http.StatusBadRequest)

		return
	}

	cmd := command.ChangeSettingsCmd{
		UserName:   user,
		Scope:      chi.URLParam(r, "scope"),
		DeviceName: r.URL.Query().Get("device"),
		Podcast:    r.URL.Query().Get("podcast"),
		Episode:    r.URL.Query().Get("episode"),
		Set:        reqData.Set,
		Remove:     reqData.Remove,
	}
	if err := u.settingsSrv.SaveSettings(ctx, &cmd); err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("SettingsResource: save settings user_name=%s scope=%s error=%q", user, cmd.Scope, err)

		return
	}

	w.WriteHeader(http.StatusOK)
}
