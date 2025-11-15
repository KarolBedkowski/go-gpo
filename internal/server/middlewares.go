package server

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
	"runtime/debug"
	"strings"
	"time"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/service"
)

func AuthenticatedOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := hlog.FromRequest(r)
		sess := session.GetSession(r)
		user := internal.SessionUser(sess)

		logger.Debug().Str("mod", "server").Str("session_user", user).Msg("authenticated only check")

		if user != "" {
			ctx := internal.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		sess.Flush()
		_ = sess.Destroy(w, r)

		w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpo\"")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}

//-------------------------------------------------------------

type authenticator struct {
	usersSrv *service.UsersSrv
}

func newAuthenticator(i do.Injector) (authenticator, error) {
	return authenticator{
		usersSrv: do.MustInvoke[*service.UsersSrv](i),
	}, nil
}

func (a authenticator) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok && password != "" && username != "" {
			ctx := r.Context()
			logger := hlog.FromRequest(r).With().Str("mod", "server").Logger()
			sess := session.GetSession(r)

			_, err := a.usersSrv.LoginUser(ctx, username, password)
			if errors.Is(err, service.ErrUnauthorized) || errors.Is(err, service.ErrUnknownUser) {
				logger.Warn().Err(err).Str("user_name", username).Msg("auth failed")
				w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpo\"")

				sess.Flush()
				_ = sess.Destroy(w, r)

				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

				return
			} else if err != nil {
				logger.Error().Err(err).Msg("login user internal error")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

				return
			}

			lloger := logger.With().Str("user_name", username).Logger()
			ctx = lloger.WithContext(ctx)

			lloger.Info().Msgf("user authenticated")

			r = r.WithContext(internal.ContextWithUser(ctx, username))
			_ = sess.Set("user", username)
		}

		next.ServeHTTP(w, r)
	})
}

//-------------------------------------------------------------

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
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if shouldSkipLogRequest(request) {
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
			loglevel := zerolog.InfoLevel
			if lrw.status >= 400 && lrw.status != 404 {
				loglevel = zerolog.WarnLevel
			}

			llog.WithLevel(loglevel).
				Str("uri", request.RequestURI).
				Int("status", lrw.status).
				Int("size", lrw.size).
				Dur("duration", time.Since(start)).
				Msg("webhandler: request finished")
		}()

		next.ServeHTTP(lrw, request)
	})
}

//-------------------------------------------------------------

// newFullLogMiddleware create new logging middleware.
func newFullLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if shouldSkipLogRequest(request) {
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
			Interface("headers", request.Header).
			Msg("webhandler: request start")

		var reqBody, respBody bytes.Buffer

		request.Body = io.NopCloser(io.TeeReader(request.Body, &reqBody))
		lrw := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

		lrw.Tee(&respBody)

		defer func() {
			if shouldLogRequestBody(request) {
				llog.Debug().
					Interface("req-headers", request.Header).
					Msg("request data: " + reqBody.String())
				llog.Debug().
					Interface("resp-headers", lrw.Header()).
					Msg("response data: " + respBody.String())
			}

			loglevel := zerolog.InfoLevel
			if lrw.Status() >= 400 && lrw.Status() != 404 {
				loglevel = zerolog.ErrorLevel
			}

			llog.WithLevel(loglevel).
				Str("uri", request.RequestURI).
				Int("status", lrw.Status()).
				Int("size", lrw.BytesWritten()).
				Dur("duration", time.Since(start)).
				Msg("webhandler: request finished")
		}()

		next.ServeHTTP(lrw, request)
	})
}

//-------------------------------------------------------------

// shouldSkipLogRequest determine which request should not be logged.
func shouldSkipLogRequest(request *http.Request) bool {
	path := request.URL.Path

	return strings.HasPrefix(path, "/metrics") || strings.HasPrefix(path, "/debug")
}

func shouldLogRequestBody(request *http.Request) bool {
	path := request.URL.Path

	return strings.HasPrefix(path, "/api")
}

//-------------------------------------------------------------

type logMiddleware func(http.Handler) http.Handler

func newLogMiddleware(i do.Injector) (logMiddleware, error) {
	cfg := do.MustInvoke[*Configuration](i)

	if cfg.DebugFlags.HasFlag(config.DebugMsgBody) {
		return newFullLogMiddleware, nil
	}

	return newSimpleLogMiddleware, nil
}

//-------------------------------------------------------------

func newRecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func(ctx context.Context) {
			rec := recover()
			if rec == nil {
				return
			}

			logger := log.Ctx(ctx).With().Str("mod", "server").Str("stack", string(debug.Stack())).Logger()

			switch t := rec.(type) {
			case error:
				logger.Error().Err(t).Msg("panic when handling request")

				if errors.Is(t, http.ErrAbortHandler) {
					panic(t)
				}
			case string:
				logger.Error().Str("err", t).Msg("panic when handling request")
			default:
				logger.Error().Str("err", "unknown error").Msg("panic when handling request")
			}

			if req.Header.Get("Connection") != "Upgrade" {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}(req.Context())

		next.ServeHTTP(w, req)
	})
}

//-------------------------------------------------------------

type sessionMiddleware func(http.Handler) http.Handler

func newSessionMiddleware(i do.Injector) (sessionMiddleware, error) {
	db := do.MustInvoke[*db.Database](i)
	repo := do.MustInvoke[repository.SessionRepository](i)

	session.RegisterFn("db", func() session.Provider {
		return service.NewSessionProvider(db, repo, sessionMaxLifetime)
	})

	sess, err := session.Sessioner(session.Options{
		Provider:       "db",
		ProviderConfig: "./tmp/",
		CookieName:     "sessionid",
		// Secure:         true,
		SameSite:    http.SameSiteLaxMode,
		Maxlifetime: sessionMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("start session manager error: %w", err)
	}

	return sess, nil
}
