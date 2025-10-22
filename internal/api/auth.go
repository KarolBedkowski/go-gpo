package api

// auth.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"errors"
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
	r.Post("/{user}/login.json", wrap(ar.login))
	r.Post("/{user}/logout.json", wrap(ar.logout))

	return r
}

func (ar *authResource) login(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := internal.ContextUser(ctx)

	logger.Debug().Str("sessionid", sess.ID()).Msg("login")

	if u := sessionUser(sess); u != "" {
		if u == user {
			logger.Debug().Msgf("user match session user %q", u)
			w.WriteHeader(http.StatusOK)
		} else {
			logger.Info().Msgf("user not match session user %q", u)
			w.WriteHeader(http.StatusBadRequest)
		}

		return
	}

	username, password, ok := r.BasicAuth()
	if !ok || user == "" || password == "" || username != user {
		logger.Info().Str("username", username).Msg("bad basic auth")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	u, err := ar.users.LoginUser(ctx, username, password)
	switch {
	case errors.Is(err, service.ErrUnauthorized) || errors.Is(err, service.ErrUnknownUser):
		logger.Info().Str("username", username).Msgf("no auth; user: %v", u)
		w.WriteHeader(http.StatusUnauthorized)

		return
	case err != nil:
		logger.Warn().Err(err).Str("username", username).Msgf("login user error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	logger.Info().Str("user", username).Msg("user authenticated")

	_ = sess.Set("user", username)

	// 	expire := time.Now().Add(5 * time.Minute)
	// 	cookie := http.Cookie{Name: "sessionid", Value: "", Path: "/", SameSite: http.SameSiteLaxMode, Expires: expire}

	// http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
}

func (*authResource) logout(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := internal.ContextUser(ctx)
	username := sessionUser(sess)

	logger.Info().Str("user", user).Msg("logout user")

	if username != "" && user != username {
		logger.Info().Str("user", user).Msgf("logout user error; session user %q not match user", username)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	_ = sess.Destroy(w, r)

	w.WriteHeader(http.StatusOK)
}
