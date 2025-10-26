package api

// auth.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpodder/internal"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

type authResource struct {
	cfg   *Configuration
	users *service.Users
}

func (ar *authResource) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post(`/{user:[\w.+-]}/login.json`, wrap(ar.login))
	r.Post(`/{user:[\w.+-]}/logout.json`, wrap(ar.logout))

	return r
}

func (ar *authResource) login(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := internal.ContextUser(ctx)

	switch u := sessionUser(sess); u {
	case "":
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	case user:
		w.WriteHeader(http.StatusOK)
	default:
		logger.Info().Msgf("user not match session user %q", u)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (*authResource) logout(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := internal.ContextUser(ctx)
	username := sessionUser(sess)

	logger.Info().Str("user", user).Msg("logout user")

	if username != "" && user != username {
		logger.Info().Str("user", user).Msgf("logout user error; session user %q not match user", username)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	_ = sess.Destroy(w, r)

	w.WriteHeader(http.StatusOK)
}
