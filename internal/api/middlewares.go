package api

//
// middlewares.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpodder/internal"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

func AuthenticatedOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := hlog.FromRequest(r)
		sess := session.GetSession(r)
		user := sessionUser(sess)

		logger.Debug().Str("session_user", user).Msg("AuthenticatedOnly")

		if user != "" {
			ctx := internal.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		_ = sess.Destroy(w, r)

		w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpodder\"")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}

type authenticator struct {
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

				_ = sess.Destroy(w, r)

				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

				return
			} else if err != nil {
				panic(err)
			}

			lloger := logger.With().Str("username", username).Logger()
			ctx = lloger.WithContext(ctx)

			lloger.Debug().Msgf("user authenticated")

			r = r.WithContext(internal.ContextWithUser(ctx, user.Name))
			_ = sess.Set("user", user.Name)
		}

		next.ServeHTTP(w, r)
	})
}

type logResponseWriter struct {
	http.ResponseWriter // compose original http.ResponseWriter

	status int // http status
	size   int // response size
}

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

func newSimpleLogMiddleware(next http.Handler) http.Handler {
	logFn := func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/metrics") {
			next.ServeHTTP(writer, request)

			return
		}

		start := time.Now()
		ctx := request.Context()
		requestID, _ := hlog.IDFromCtx(ctx)
		llog := log.With().Logger().With().Str("req_id", requestID.String()).Logger()
		request = request.WithContext(llog.WithContext(ctx))

		llog.Info().
			Str("url", request.URL.Redacted()).
			Str("remote", request.RemoteAddr).
			Str("method", request.Method).
			Msg("webhandler: request start")

		lrw := &logResponseWriter{ResponseWriter: writer, status: 0, size: 0}

		defer func() {
			l := zerolog.InfoLevel
			if lrw.status >= 400 && lrw.status != 404 {
				l = zerolog.WarnLevel
			}

			llog.WithLevel(l).
				Str("uri", request.RequestURI).
				Int("status", lrw.status).
				Int("size", lrw.size).
				Dur("duration", time.Since(start)).
				Msg("webhandler: request finished")
		}()

		next.ServeHTTP(lrw, request)
	}

	return http.HandlerFunc(logFn)
}

// newLogMiddleware create new logging middleware.
func newLogMiddleware(next http.Handler) http.Handler {
	logFn := func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/metrics") {
			next.ServeHTTP(writer, request)

			return
		}

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
				Str("req-content-type", request.Header.Get("Content-Type")).
				Interface("req-headers", request.Header).
				Msg("request")
			llog.Debug().Str("response_body", respBody.String()).
				Str("resp-content-type", lrw.Header().Get("Content-Type")).
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
		defer func(ctx context.Context) {
			rec := recover()
			if rec == nil {
				return
			}

			logger := log.Ctx(ctx)

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
		}(req.Context())

		next.ServeHTTP(w, req)
	})
}

func checkUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger := hlog.FromRequest(req)

		user := chi.URLParam(req, "user")
		if user == "" {
			logger.Warn().Msgf("empty user in params, %q", chi.URLParam(req, "deviceid"))
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		sess := session.GetSession(req)
		if suser := sessionUser(sess); suser != "" {
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
		llogger := logger.With().Str("username", user).Logger()
		ctx = llogger.WithContext(ctx)

		llogger.Debug().Msgf("found user %q in params", user)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func checkDeviceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		deviceid := chi.URLParam(req, "deviceid")
		if deviceid == "" {
			hlog.FromRequest(req).Info().Msgf("empty deviceid")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		ctx := internal.ContextWithDevice(req.Context(), deviceid)
		logger := hlog.FromRequest(req).With().Str("deviceid", deviceid).Logger()
		ctx = logger.WithContext(ctx)

		logger.Debug().Msgf("found device %q in params", deviceid)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

type sessionMiddleware struct {
	sess func(next http.Handler) http.Handler
}

func (s *sessionMiddleware) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/metrics") {
			next.ServeHTTP(w, req)

			return
		}

		s.sess(next).ServeHTTP(w, req)
	})
}
