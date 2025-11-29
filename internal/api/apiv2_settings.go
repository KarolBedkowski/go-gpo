// updates.g
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

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
		Get(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, srvsupport.Wrap(u.getSettings))
	r.With(checkUserMiddleware).
		Post(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, srvsupport.Wrap(u.postSettings))

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
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get settings error")

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &res)
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
		logger.Debug().Err(err).Msg("decode request error")
		writeError(w, r, http.StatusBadRequest)

		return
	}

	cmd := command.ChangeSettingsCmd{
		UserName:   user,
		Scope:      chi.URLParam(r, "scope"),
		DeviceName: r.URL.Query().Get("device"),
		Podcast:    r.URL.Query().Get("podcast"),
		Episode:    r.URL.Query().Get("episode"),
	}
	if err := u.settingsSrv.SaveSettings(ctx, &cmd); err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("save settings error")

		return
	}

	w.WriteHeader(http.StatusOK)
}
