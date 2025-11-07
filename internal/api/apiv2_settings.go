// updates.g
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal"
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

	key, err := u.getKey(r)
	if err != nil {
		logger.Debug().Err(err).Str("mod", "api").Msg("get key error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	scope := chi.URLParam(r, "scope")

	res, err := u.settingsSrv.GetSettings(ctx, user, scope, key)
	if err != nil {
		if internal.CheckAndWriteError(w, r, err) {
			logger.Warn().Err(err).Str("mod", "api").Msg("get settings error")
		} else {
			logger.Debug().Err(err).Str("mod", "api").Msg("get settings error")
		}

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

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		logger.Debug().Err(err).Str("mod", "api").Msg("decode request error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	key, err := u.getKey(r)
	if err != nil {
		logger.Debug().Err(err).Str("mod", "api").Msg("get key error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	scope := chi.URLParam(r, "scope")

	if err := u.settingsSrv.SaveSettings(ctx, user, scope, key, req.Set, req.Remove); err != nil {
		if internal.CheckAndWriteError(w, r, err) {
			logger.Warn().Err(err).Str("mod", "api").Msg("save settings error")
		} else {
			logger.Debug().Err(err).Str("mod", "api").Msg("save settings error")
		}

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (u settingsResource) getKey(r *http.Request) (string, error) {
	var key string

	scope := chi.URLParam(r, "scope")
	switch scope {
	case "account":
		return "", nil
	case "device":
		key = r.URL.Query().Get("device")
	case "episode":
		key = r.URL.Query().Get("episode")
	case "podcast":
		key = r.URL.Query().Get("podcast")
	default:
		return "", fmt.Errorf("unknown scope: %q", scope) //nolint:err113
	}

	if key == "" {
		return "", errors.New("missing required keys") //nolint:err113
	}

	return key, nil
}
