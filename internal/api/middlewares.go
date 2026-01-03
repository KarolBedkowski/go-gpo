package api

//
// middlewares.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
)

//-------------------------------------------------------------

func checkUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger := hlog.FromRequest(req)

		user := chi.URLParam(req, "user")
		if user == "" {
			logger.Debug().Msg("api.CheckUser: bad request - missing or empty user")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		sess := session.GetSession(req)
		// when auth is enabled authenticator always set session user or block request to get here.
		if suser := srvsupport.SessionUser(sess); suser != "" {
			// auth enabled
			if suser != user {
				logger.Warn().Msgf("api.CheckUser: user_name=%s not match session_user=%s", user, suser)
				w.WriteHeader(http.StatusBadRequest)

				return
			}
		} else {
			// TODO: remove
			// auth disabled; put user into session
			if err := sess.Set("user", user); err != nil {
				logger.Error().Err(err).Msgf("api.CheckUser: set session for user_name=%s error=%q", user, err)
			}
		}

		ctx := common.ContextWithUser(req.Context(), user)
		// handled by authenticator
		// llogger := logger.With().Str(common.LogKeyUserName, user).Logger()
		// ctx = llogger.WithContext(ctx)

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func checkDeviceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		devicename := chi.URLParam(req, "devicename")
		if devicename == "" {
			hlog.FromRequest(req).Debug().Msg("api.CheckDevice: bad request - missing or empty devicename")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		ctx := common.ContextWithDevice(req.Context(), devicename)
		logger := hlog.FromRequest(req).With().Str(common.LogKeyDeviceID, devicename).Logger()
		ctx = logger.WithContext(ctx)

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}
