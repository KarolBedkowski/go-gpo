//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

type Configuration struct {
	cfg    *Configuration
	NoAuth bool
}

func Start(repo *repository.Repository, cfg *Configuration) {
	sess, err := session.Sessioner(session.Options{
		Provider:       "file",
		ProviderConfig: "./tmp/",
		CookieName:     "sessionid",
		// Secure:         true,
		// SameSite:       http.SameSiteLaxMode,
		// Maxlifetime: 60 * 60 * 24 * 365,
	})
	if err != nil {
		panic(err.Error())
	}

	deviceSrv := service.NewDeviceService(repo)
	subSrv := service.NewSubssService(repo)
	usersSrv := service.NewUsersService(repo)
	episodesSrv := service.NewEpisodesService(repo)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	r.Use(newLogMiddleware)
	r.Use(sess)
	r.Use(authenticator{usersSrv}.Authenticate)
	r.Use(newRecoverMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Mount("/subscriptions", (&simpleResource{cfg, repo, subSrv}).Routes())

	r.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", authResource{cfg, usersSrv}.Routes())
		r.Mount("/devices", deviceResource{cfg, deviceSrv}.Routes())
		r.Mount("/subscriptions", (&subscriptionsResource{cfg, subSrv}).Routes())
		r.Mount("/episodes", (&episodesResource{cfg, episodesSrv}).Routes())
		r.Mount("/updates", (&updatesResource{cfg, subSrv, episodesSrv}).Routes())
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("go-gpodder"))
	})

	logRoutes(r)

	http.ListenAndServe(":3000", r)
}

func AuthenticatedOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := hlog.FromRequest(r)
		sess := session.GetSession(r)
		user := sessionUser(sess)

		logger.Debug().Interface("session_user", user).Msg("AuthenticatedOnly")

		if user != "" {
			ctx := service.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		sess.Destroy(w, r)
		w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpodder\"")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
}

type authenticator struct {
	// TODO: service
	usersSrv *service.Users
}

func (a authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok && password != "" && username != "" {
			ctx := r.Context()
			logger := hlog.FromRequest(r)
			sess := session.GetSession(r)

			user, err := a.usersSrv.LoginUser(ctx, username, password)
			if errors.Is(err, service.ErrUnauthorized) || errors.Is(err, service.ErrUnknownUser) {
				logger.Info().Err(err).Str("username", username).Msgf("auth failed; user: %v", user)
				w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpodder\"")
				sess.Destroy(w, r)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

				return
			} else if err != nil {
				panic(err)
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

		llog.Info().
			Str("url", request.URL.Redacted()).
			Str("remote", request.RemoteAddr).
			Str("method", request.Method).
			// Strs("agent", request.Header["User-Agent"]).
			// Interface("headers", request.Header).
			Msg("webhandler: request start")

		var reqBody bytes.Buffer
		request.Body = io.NopCloser(io.TeeReader(request.Body, &reqBody))

		lrw := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)
		// lrw := logResponseWriter{ResponseWriter: writer, status: 0, size: 0}

		var respBody bytes.Buffer
		lrw.Tee(&respBody)

		defer func() {
			llog.Debug().Str("request_body", reqBody.String()).
				Str("req-content-type", request.Header.Get("content-type")).
				Interface("req-headers", request.Header).
				Msg("request")
			llog.Debug().Str("response_body", respBody.String()).
				Str("resp-content-type", lrw.Header().Get("content-type")).
				Interface("resp-headers", lrw.Header()).
				Msg("response")

			if lrw.Status() >= 400 && lrw.Status() != 404 {
				llog.Error().
					Str("uri", request.RequestURI).
					Interface("req_headers", request.Header).
					Interface("resp_header", lrw.Header()).
					Int("status", lrw.Status()).
					Int("size", lrw.BytesWritten()).
					Dur("duration", time.Since(start)).
					Msg("webhandler: request finished")

				return
			}

			llog.Debug().
				Str("uri", request.RequestURI).
				// Interface("resp_header", lrw.ResponseWriter.Header()).
				Int("status", lrw.Status()).
				Int("size", lrw.BytesWritten()).
				Dur("duration", time.Since(start)).
				Msg("webhandler: request finished")
		}()

		next.ServeHTTP(lrw, request)
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

func sessionUser(store session.Store) string {
	log.Debug().Interface("session", store).Msg("session")
	suserint := store.Get("user")
	if username, ok := suserint.(string); ok {
		return username
	}

	return ""
}

func checkUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if user := chi.URLParam(req, "user"); user != "" {
			if suser := service.ContextUser(req.Context()); suser != user {
				logger := hlog.FromRequest(req)
				logger.Warn().Msgf("user %q not match session user: %q", user, suser)
				w.WriteHeader(http.StatusBadRequest)

				return
			}
		}

		next.ServeHTTP(w, req)
	})
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
