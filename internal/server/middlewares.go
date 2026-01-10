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
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
)

func AuthenticatedOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := hlog.FromRequest(r)
		sess := session.GetSession(r)
		user := srvsupport.SessionUser(sess)

		logger.Debug().Str("session_user", user).Str("sid", sess.ID()).
			Msgf("AuthenticatedOnly: check user_name=%s sid=%s", user, sess.ID())

		if user != "" {
			next.ServeHTTP(w, r)

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
		username, password, basicAuthOk := r.BasicAuth()
		sess := session.GetSession(r)
		sessionuser, _ := sess.Get("user").(string)
		logger := hlog.FromRequest(r).With().
			Str("sid", sess.ID()).Str(common.LogKeyUserName, common.Coalesce(username, sessionuser)).Logger()
		ctx := logger.WithContext(r.Context())

		common.NewRegion(ctx, "authenticator handle").End()

		if sessionuser == "" && !basicAuthOk {
			// no valid session, no auth, continue to next handler
			next.ServeHTTP(w, r.WithContext(common.ContextWithUser(ctx, username)))

			return
		}

		// if session is valid and (there is no new auth or there is auth but username is not changed) - continue
		if sessionuser != "" && (!basicAuthOk || username == sessionuser) {
			next.ServeHTTP(w, r.WithContext(common.ContextWithUser(ctx, sessionuser)))

			return
		}

		switch _, err := a.usersSrv.LoginUser(ctx, username, password); {
		case err == nil:
			// no error login/check user - continue
			logger.Info().Str(common.LogKeyAuthResult, common.LogAuthResultSuccess).
				Msgf("Authenticator: user authenticated user_name=%s", username)

			_ = sess.Set("user", username)

			common.TraceLazyPrintf(ctx, "AuthenticationError: user authenticated")
			next.ServeHTTP(w, r.WithContext(common.ContextWithUser(ctx, username)))
		case aerr.HasTag(err, common.AuthenticationError):
			logger.Info().Err(err).Str(common.LogKeyUserName, username).
				Str(common.LogKeyAuthResult, common.LogAuthResultFailed).
				Msgf("Authenticator: user authentication failed user_name=%s: %s", username, err)
			// destroy session
			sess.Flush()
			_ = sess.Destroy(w, r)

			common.TraceLazyPrintf(ctx, "Authenticator: auth failed")
			w.Header().Add("WWW-Authenticate", "Basic realm=\"go-gpo\"")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		default:
			common.TraceErrorLazyPrintf(ctx, "Authenticator: auth error")
			logger.Error().Err(err).Msgf("Authenticator: internal error user_name=%s error=%q", username, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
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

		start := time.Now().UTC()
		ctx := request.Context()
		requestID, _ := hlog.IDFromCtx(ctx)
		llog := log.With().Str(common.LogKeyReqID, requestID.String()).Logger()
		request = request.WithContext(llog.WithContext(ctx))
		user, _, _ := request.BasicAuth()

		llog.Info().
			Str("url", request.URL.Redacted()).
			Str("remote", request.RemoteAddr).
			Str("method", request.Method).
			Str("req_user", user).
			Msgf("Server: request start method=%s url=%s", request.Method, request.URL.Redacted())

		lrw := &logResponseWriter{ResponseWriter: writer, status: 0, size: 0}

		defer func() {
			loglevel, dloglevel := mapStatusToLogLevel(lrw.status)

			if e := llog.WithLevel(dloglevel); e.Enabled() {
				e.Interface(common.LogKeyRequestHeaders, filterHeaders(request.Header)).
					Msg("Server: request headers")
				llog.WithLevel(dloglevel).Interface(common.LogKeyResponseHeaders, lrw.Header()).
					Msg("Server: response headers")
			}

			llog.WithLevel(loglevel).
				Str("url", request.URL.Redacted()).
				Int("status", lrw.status).
				Int("size", lrw.size).
				Dur("duration", time.Since(start)).
				Str("req_user", user).
				Msgf("Server: request finished method=%s url=%s status=%d", request.Method, request.URL.Redacted(), lrw.status)
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

		start := time.Now().UTC()
		ctx := request.Context()
		requestID, _ := hlog.IDFromCtx(ctx)
		llog := log.With().Str(common.LogKeyReqID, requestID.String()).Logger()
		request = request.WithContext(llog.WithContext(ctx))
		user, _, _ := request.BasicAuth()

		llog.Info().
			Str("url", request.URL.Redacted()).
			Str("remote", request.RemoteAddr).
			Str("method", request.Method).
			Str("req_user", user).
			Msgf("Server: request start method=%s url=%s", request.Method, request.URL.Redacted())

		var reqBody, respBody bytes.Buffer

		request.Body = io.NopCloser(io.TeeReader(request.Body, &reqBody))
		lrw := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

		lrw.Tee(&respBody)

		defer func() {
			if shouldLogRequestBody(request) {
				llog.Debug().Msg("request body: " + reqBody.String())
				llog.Debug().Msg("response body: " + respBody.String())
			}

			loglevel, dloglevel := mapStatusToLogLevel(lrw.Status())

			if e := llog.WithLevel(dloglevel); e.Enabled() {
				e.Interface(common.LogKeyRequestHeaders, filterHeaders(request.Header)).
					Msg("Server: request headers")
				llog.WithLevel(dloglevel).Interface(common.LogKeyResponseHeaders, lrw.Header()).
					Msg("Server: response headers")
			}

			llog.WithLevel(loglevel).
				Str("url", request.RequestURI).
				Int("status", lrw.Status()).
				Int("size", lrw.BytesWritten()).
				Str("req_user", user).
				Dur("duration", time.Since(start)).
				Msgf("Server: request finished method=%s url=%s status=%d",
					request.Method, request.URL.Redacted(), lrw.Status())
		}()

		next.ServeHTTP(lrw, request)
	})
}

//-------------------------------------------------------------

// newVerySimpleLogMiddleware create basic log middleware that log only result request.
func newVerySimpleLogMiddleware(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			start := time.Now().UTC()
			ctx := request.Context()
			requestID, _ := hlog.IDFromCtx(ctx)
			llog := log.With().Str(common.LogKeyReqID, requestID.String()).Logger()
			request = request.WithContext(llog.WithContext(ctx))
			lrw := &logResponseWriter{ResponseWriter: writer, status: 0, size: 0}

			defer func() {
				loglevel, _ := mapStatusToLogLevel(lrw.status)
				llog.WithLevel(loglevel).
					Str("url", request.URL.Redacted()).
					Int("status", lrw.status).
					Int("size", lrw.size).
					Dur("duration", time.Since(start)).
					Msgf(name+": request finished method=%s url=%s status=%d", request.Method, request.URL.Redacted(), lrw.status)
			}()

			next.ServeHTTP(lrw, request)
		})
	}
}

//-------------------------------------------------------------

// shouldSkipLogRequest determine which request should not be logged.
func shouldSkipLogRequest(request *http.Request) bool {
	path := request.URL.Path

	return strings.HasPrefix(path, "/metrics") || strings.HasPrefix(path, "/debug") ||
		path == "/favicon.ico" || strings.HasPrefix(path, "/web/static/")
}

func shouldLogRequestBody(request *http.Request) bool {
	path := request.URL.Path

	return strings.HasPrefix(path, "/api")
}

//-------------------------------------------------------------

type logMiddleware func(http.Handler) http.Handler

func newLogMiddleware(i do.Injector) (logMiddleware, error) {
	cfg := do.MustInvoke[*config.ServerConf](i)

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

			common.TraceErrorLazyPrintf(ctx, "RecoveryMW: panic: %v", rec)

			logger := log.Ctx(ctx).With().Str("stack", string(debug.Stack())).Logger()

			switch t := rec.(type) {
			case error:
				logger.Error().Err(t).Msgf("RecoveryMW: panic when handling request: %s", rec)

				if errors.Is(t, http.ErrAbortHandler) {
					panic(t)
				}
			case string:
				logger.Error().Str("err", t).Msgf("RecoveryMW: panic when handling request: %s", rec)
			default:
				logger.Error().
					Str("err", fmt.Sprintf("%v", rec)).
					Msg("RecoveryMW: panic when handling request: unknown error")
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
	dbi := do.MustInvoke[repository.Database](i)
	repo := do.MustInvoke[repository.Sessions](i)
	cfg := do.MustInvoke[*config.ServerConf](i)

	session.RegisterFn("db", func() session.Provider {
		return service.NewSessionProvider(dbi, repo, sessionMaxLifetime)
	})

	sess, err := session.Sessioner(session.Options{
		Provider:       "db",
		ProviderConfig: "./tmp/",
		CookieName:     "sessionid",
		SameSite:       http.SameSiteLaxMode,
		Maxlifetime:    int64(sessionMaxLifetime.Seconds()),
		Secure:         cfg.MainServer.UseSecureCookie(),
		CookiePath:     cfg.MainServer.WebRoot,
	})
	if err != nil {
		return nil, aerr.Wrapf(err, "start session manager failed")
	}

	return sess, nil
}

//-------------------------------------------------------------

// filterHeaders remove sensitive data from request header for logging.
func filterHeaders(h http.Header) http.Header {
	if h.Get("Authorization") == "" {
		return h
	}

	h = h.Clone()
	h.Set("Authorization", "<redacted>")

	return h
}

// mapStatusToLogLevel get http status and return level for regular logs and level for debug logs.
func mapStatusToLogLevel(status int) (zerolog.Level, zerolog.Level) {
	switch {
	case status == http.StatusInternalServerError:
		return zerolog.ErrorLevel, zerolog.ErrorLevel
	case status >= 500: //nolint:mnd
		return zerolog.WarnLevel, zerolog.WarnLevel
	default:
		return zerolog.InfoLevel, zerolog.DebugLevel
	}
}

//-------------------------------------------------------------

func newAuthDebugMiddleware(c *config.ServerConf) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := zerolog.Ctx(r.Context())

			if allow, _ := c.AuthMgmtRequest(r); allow {
				log.Debug().Msgf("AuthDebug: access to url=%q from remote=%q allowed", r.URL.Redacted(), r.RemoteAddr)
				next.ServeHTTP(w, r)
			} else {
				log.Debug().Msgf("AuthDebug: access to url=%q from remote=%q forbidden", r.URL.Redacted(), r.RemoteAddr)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}
		})
	}
}
