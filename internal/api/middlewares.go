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
		suser := srvsupport.SessionUser(sess)

		switch {
		case suser == "":
			logger.Warn().Msgf("api.CheckUser: missing authentication for user_name=%s", user)
			w.WriteHeader(http.StatusForbidden)
		case suser != user:
			logger.Warn().Msgf("api.CheckUser: user_name=%s not match session_user=%s", user, suser)
			w.WriteHeader(http.StatusBadRequest)
		default:
			ctx := common.ContextWithUser(req.Context(), user)
			next.ServeHTTP(w, req.WithContext(ctx))
		}
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
