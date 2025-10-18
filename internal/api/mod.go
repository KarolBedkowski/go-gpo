//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpodder/internal/repository"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

func Start(repo *repository.Repository) {
	sess, err := session.Sessioner(session.Options{
		Provider:       "file",
		ProviderConfig: "./tmp/",
		CookieName:     "sessionid",
		SameSite:       http.SameSiteLaxMode,
		Maxlifetime:    5 * 60,
	})
	if err != nil {
		panic(err.Error())
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(sess)
	r.Use(authenticator{repo}.Authenticate)
	r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	r.Use(middleware.RealIP)
	r.Use(newLogMiddleware)
	// r.Use(httplog.RequestLogger(logger, &httplog.Options{
	// 	Level: slog.LevelDebug,
	// 	RecoverPanics: true,
	// 	LogRequestHeaders:  []string{"Cookie", "Authentication"},
	// 	LogResponseHeaders: []string{"Cookie"},
	// 	LogRequestBody:  isDebugHeaderSet,
	// 	LogResponseBody: isDebugHeaderSet,
	// }))

	r.Use(newRecoverMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))

	deviceSrv := service.NewDeviceService(repo)
	subSrv := service.NewSubssService(repo)
	usersSrv := service.NewUsersService(repo)
	episodesSrv := service.NewEpisodesService(repo)

	r.Mount("/subscriptions", (&simpleResource{repo, subSrv}).Routes())

	r.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", authResource{usersSrv}.Routes())
		r.Mount("/devices", deviceResource{deviceSrv}.Routes())
		r.Mount("/subscriptions", (&subscriptionsResource{subSrv}).Routes())
		r.Mount("/episodes", (&episodesResource{episodesSrv}).Routes())
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	logRoutes(r)

	http.ListenAndServe(":3000", r)
}

func AuthenticatedOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.GetSession(r)
		logger := hlog.FromRequest(r)
		logger.Debug().Interface("session", sess.Get("user")).Msg("AuthenticatedOnly")

		if suser, ok := sess.Get("user").(string); ok && suser != "" {
			ctx := context.WithValue(r.Context(), "user", suser)
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		_ = sess.Set("user", "")

		w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpodder\"")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
}

func userFromContext(ctx context.Context) string {
	suser, ok := ctx.Value("user").(string)
	if ok {
		return suser
	}

	return ""
}

type authenticator struct {
	// TODO: service
	repo *repository.Repository
}

func (a authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok && password != "" && username != "" {
			ctx := r.Context()
			logger := hlog.FromRequest(r)

			user, err := a.repo.GetUser(ctx, username)
			if err != nil {
				panic(err)
			}

			sess := session.GetSession(r)

			if user == nil || !user.CheckPassword(password) {
				logger.Info().Str("username", username).Msgf("auth failed; user: %v", user)
				w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpodder\"")
				_ = sess.Set("user", "")
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

				return
			}

			logger.Debug().Str("username", username).Msgf("user authenticated")
			ctx = context.WithValue(ctx, "user", user.Name)
			r = r.WithContext(ctx)

			_ = sess.Set("user", user.Name)
		}

		next.ServeHTTP(w, r)
	})
}

type (
	// our http.ResponseWriter implementation.
	logResponseWriter struct {
		http.ResponseWriter // compose original http.ResponseWriter

		status int // http status
		size   int // response size
	}
)

func (r *logResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b) // write response using original http.ResponseWriter
	r.size += size                         // capture size

	if err != nil {
		return size, fmt.Errorf("write response error: %w", err)
	}

	return size, nil
}

func (r *logResponseWriter) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)

	r.status = status
}

// newLogMiddleware create new logging middleware.
func newLogMiddleware(next http.Handler) http.Handler {
	logFn := func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		ctx := request.Context()
		requestID, _ := hlog.IDFromCtx(ctx)
		llog := log.With().Logger().With().Str("req_id", requestID.String()).Logger()
		request = request.WithContext(llog.WithContext(ctx))
		sess := session.GetSession(request)

		llog.Info().
			Str("url", request.URL.Redacted()).
			Str("remote", request.RemoteAddr).
			Str("method", request.Method).
			// Strs("agent", request.Header["User-Agent"]).
			// Interface("headers", request.Header).
			Str("sessionid", sess.ID()).
			Msg("webhandler: request start")

		lrw := logResponseWriter{ResponseWriter: writer, status: 0, size: 0}

		next.ServeHTTP(&lrw, request)

		if lrw.status >= 400 && lrw.status != 404 {
			llog.Error().
				Str("uri", request.RequestURI).
				Interface("req_headers", request.Header).
				Interface("resp_header", lrw.ResponseWriter.Header()).
				Int("status", lrw.status).
				Int("size", lrw.size).
				Dur("duration", time.Since(start)).
				Msg("webhandler: request finished")

			return
		}

		llog.Debug().
			Str("uri", request.RequestURI).
			// Interface("resp_header", lrw.ResponseWriter.Header()).
			Int("status", lrw.status).
			Int("size", lrw.size).
			Dur("duration", time.Since(start)).
			Msg("webhandler: request finished")
	}

	return http.HandlerFunc(logFn)
}

func newRecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() { //nolint:contextcheck
			rec := recover()

			if rec == nil {
				return
			}

			logger := log.Ctx(req.Context())
			switch t := rec.(type) {
			case error:
				if errors.Is(t, http.ErrAbortHandler) {
					panic(t)
				}

				logger.Error().Err(t).Msg("panic when handling request")
			case string:
				logger.Error().Str("err", t).Msg("panic when handling request")
			default:
				logger.Error().Str("err", "unknown error").Msg("panic when handling request")
			}

			if req.Header.Get("Connection") != "Upgrade" {
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}()

		next.ServeHTTP(w, req)
	})
}

func userFromsession(store session.Store) string {
	log.Debug().Interface("session", store).Msg("session")
	suserint := store.Get("user")
	if username, ok := suserint.(string); ok {
		return username
	}

	return ""
}

func logRoutes(r chi.Routes) {
	walkFunc := func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		route = strings.ReplaceAll(route, "/*/", "/")
		log.Debug().Msgf("ROUTE: %s %s", method, route)
		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		log.Error().Err(err).Msg("routers walk error")
	}
}
