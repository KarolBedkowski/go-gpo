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
	"gitlab.com/kabes/go-gpo/internal"
)

//-------------------------------------------------------------

func checkUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger := hlog.FromRequest(req)

		user := chi.URLParam(req, "user")
		if user == "" {
			logger.Debug().Msgf("empty user")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		sess := session.GetSession(req)
		if suser := internal.SessionUser(sess); suser != "" {
			// auth enabled
			if suser != user {
				logger.Warn().Msgf("user %q not match session user: %q", user, suser)
				w.WriteHeader(http.StatusBadRequest)

				return
			}
		} else {
			// auth disabled; put user into session
			sess.Set("user", user)
		}

		ctx := internal.ContextWithUser(req.Context(), user)
		llogger := logger.With().Str("user_name", user).Logger()
		ctx = llogger.WithContext(ctx)

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func checkDeviceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		deviceid := chi.URLParam(req, "deviceid")
		if deviceid == "" {
			hlog.FromRequest(req).Debug().Msgf("empty deviceid")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		ctx := internal.ContextWithDevice(req.Context(), deviceid)
		logger := hlog.FromRequest(req).With().Str("device_id", deviceid).Logger()
		ctx = logger.WithContext(ctx)

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}
