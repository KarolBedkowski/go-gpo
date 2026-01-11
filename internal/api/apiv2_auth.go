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
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
)

// authResource handle request to /api/2/auth (login and logout) methods.
type authResource struct{}

func newAuthResource(_ do.Injector) (authResource, error) {
	return authResource{}, nil
}

func (ar authResource) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post(`/{user:[\w.+-]}/login.json`, srvsupport.WrapNamed(ar.login, "api_auth_login"))
	r.Post(`/{user:[\w.+-]}/logout.json`, srvsupport.WrapNamed(ar.logout, "api_auth_logout"))

	return r
}

func (ar authResource) login(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := common.ContextUser(ctx)

	switch u := srvsupport.SessionUser(sess); u {
	case "":
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	case user:
		w.WriteHeader(http.StatusOK)
	default:
		logger.Info().Str(common.LogKeyUserName, user).
			Msgf("AuthResource: user to login %q not match session user %q", user, u)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (ar authResource) logout(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := common.ContextUser(ctx)
	username := srvsupport.SessionUser(sess)

	logger.Info().Str(common.LogKeyUserName, user).Msgf("SimpleResource: logout user")

	if username != "" && user != username {
		logger.Info().Str(common.LogKeyUserName, user).
			Msgf("SimpleResource: logout user error; session user %q not match user", username)
		writeError(w, r, http.StatusBadRequest)

		return
	}

	sess.Flush()
	_ = sess.Destroy(w, r)

	w.WriteHeader(http.StatusOK)
}
