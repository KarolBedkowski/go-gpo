// auth.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"errors"
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

type authResource struct {
	users *service.Users
}

func (ar authResource) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{user}/login.json", ar.login)
	r.Post("/{user}/logout.json", ar.logout)

	return r
}

func (a *authResource) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	paramUser := chi.URLParam(r, "user")
	sess := session.GetSession(r)

	logger.Debug().Str("sessionid", sess.ID()).Msg("login")

	switch u := userFromsession(sess); u {
	case "":
		// not logged; continue
	case paramUser:
		logger.Debug().Msgf("user match session user %q", u)
		w.WriteHeader(http.StatusOK)
		return
	default:
		logger.Info().Msgf("user not match session user %q", u)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username, password, ok := r.BasicAuth()
	if !ok || paramUser == "" || password == "" || username != paramUser {
		logger.Info().Str("username", username).Msg("bad basic auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := a.users.LoginUser(ctx, username, password)
	switch {
	case errors.Is(err, service.ErrUnauthorized) || errors.Is(err, service.ErrUnknownUser):
		logger.Info().Str("username", username).Msgf("no auth; user: %v", user)
		w.WriteHeader(http.StatusUnauthorized)

		return
	case err != nil:
		logger.Warn().Err(err).Str("username", username).Msgf("login user error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	logger.Info().Str("user", username).Msg("user authenticated")

	sess.Set("user", username)

	// 	expire := time.Now().Add(5 * time.Minute)
	// 	cookie := http.Cookie{Name: "sessionid", Value: "", Path: "/", SameSite: http.SameSiteLaxMode, Expires: expire}

	// http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
}

func (authResource) logout(w http.ResponseWriter, r *http.Request) {
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")
	sess := session.GetSession(r)
	username := userFromsession(sess)

	logger.Info().Str("user", user).Msg("logout user")

	if username != "" && user != username {
		logger.Info().Str("user", user).Msgf("logout user error; session user %q not match user", username)
		w.WriteHeader(400)

		return
	}

	sess.Destroy(w, r)
	w.WriteHeader(200)
}
