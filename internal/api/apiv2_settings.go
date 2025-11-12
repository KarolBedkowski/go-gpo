// updates.g
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type settingsResource struct {
	settingsSrv *service.Settings
}

func newSettingsResource(i do.Injector) (settingsResource, error) {
	return settingsResource{
		settingsSrv: do.MustInvoke[*service.Settings](i),
	}, nil
}

func (u settingsResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, internal.Wrap(u.getSettings))
	r.With(checkUserMiddleware).
		Post(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, internal.Wrap(u.setSettings))

	return r
}

func (u settingsResource) getSettings(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)

	key, err := newKey(user, r)
	if err != nil {
		logger.Debug().Err(err).Str("mod", "api").Msg("bad request parameters")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	res, err := u.settingsSrv.GetSettings(ctx, &key)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get settings error")

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &res)
}

func (u settingsResource) setSettings(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)

	var req struct {
		Set    map[string]string `json:"set"`
		Remove []string          `json:"remove"`
	}

	key, err := newKey(user, r)
	if err != nil {
		logger.Debug().Err(err).Str("mod", "api").Msg("bad request parameters")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		logger.Debug().Err(err).Str("mod", "api").Msg("decode request error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	// combine set and remove - add empty value for deleted string.
	settings := req.Set
	if settings == nil {
		settings = make(map[string]string)
	}

	for _, k := range req.Remove {
		settings[k] = ""
	}

	if err := u.settingsSrv.SaveSettings(ctx, &key, settings); err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("save settings error")

		return
	}

	w.WriteHeader(http.StatusOK)
}

func newKey(user string, r *http.Request) (model.SettingsKey, error) {
	//nolint:wrapcheck
	return model.NewSettingsKey(user,
		chi.URLParam(r, "scope"),
		r.URL.Query().Get("device"),
		r.URL.Query().Get("podcast"),
		r.URL.Query().Get("episode"),
	)
}
