// updates.g
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"net/http"

	"gitlab.com/kabes/go-gpodder/internal"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
)

type settingsResource struct {
	cfg          *Configuration
	settingsServ *service.Settings
}

func (u *settingsResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !u.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
	}

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, wrap(u.getSettings))
	r.With(checkUserMiddleware).
		Post(`/{user:[\w+.-]+}/{scope:[a-z]+}.json`, wrap(u.setSettings))

	return r
}

func (u *settingsResource) getSettings(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)

	var (
		res map[string]string
		err error
		key string
	)

	scope := chi.URLParam(r, "scope")
	switch scope {
	case "account":
	case "device":
		key = r.URL.Query().Get("device")
	case "episode":
		key = r.URL.Query().Get("episode")
	default:
		logger.Info().Msgf("unknown scope: %q", scope)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	res, err = u.settingsServ.GetSettings(ctx, user, scope, key)
	if err != nil {
		logger.Info().Err(err).Str("scope", "scope").Msgf("get settings error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &res)
}

func (u *settingsResource) setSettings(
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
		logger.Info().Err(err).Msg("decode request error")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	var key string

	scope := chi.URLParam(r, "scope")
	switch scope {
	case "account":
	case "device":
		key = r.URL.Query().Get("device")
	case "episode":
		key = r.URL.Query().Get("episode")
	default:
		logger.Info().Msgf("unknown scope: %q", scope)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err := u.settingsServ.SaveSettings(ctx, user, scope, key, req.Set, req.Remove); err != nil {
		logger.Info().Err(err).Str("scope", "scope").Msgf("save settings error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	w.WriteHeader(http.StatusOK)
}
