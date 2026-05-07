package web

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type authPages struct {
	renderer *nt.Renderer
	usersSrv *service.UsersSrv
	webroot  string
}

func newAuthPages(i do.Injector) (authPages, error) {
	return authPages{
		renderer: do.MustInvoke[*nt.Renderer](i),
		usersSrv: do.MustInvoke[*service.UsersSrv](i),
		webroot:  do.MustInvokeNamed[string](i, "server.webroot"),
	}, nil
}

func (a authPages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/login", srvsupport.WrapNamed(a.login, "web_login"))
	r.Post("/login", srvsupport.WrapNamed(a.loginPost, "web_login"))
	r.Get("/logout", srvsupport.WrapNamed(a.logout, "web_logout"))

	return r
}

func (a authPages) login(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := common.ContextUser(ctx)

	logger.Info().Str(common.LogKeyUserName, user).Msgf("web: login user")

	sess.Flush()
	_ = sess.Destroy(w, r)
	_, _ = sess.RegenerateID(w, r)

	a.renderer.WriteSimplePage(w, nt.LoginPage{})
}

func (a authPages) loginPost(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)

	logger.Info().Msgf("web: do login user")

	const maxBody = 1024 * 1024 // 1k

	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseForm(); err != nil {
		logger.Error().Err(err).Msgf("web.Auth: do login bad request - parse form error=%q", err)
		a.renderer.WriteError(ctx, w, err)

		return
	}

	username := r.Form.Get("login")
	password := r.Form.Get("password")

	if username == "" || password == "" {
		a.renderer.WriteSimplePage(w, nt.LoginPage{})

		return
	}

	switch _, err := a.usersSrv.LoginUser(ctx, username, password); {
	case err == nil:
		// no error login/check user - continue
		_ = sess.Set("user", username)
		_ = sess.Release()

		logger.Info().Str(common.LogKeyAuthResult, common.LogAuthResultSuccess).
			Msgf("web.Auth: user authenticated user_name=%s", username)

		http.Redirect(w, r, a.webroot+"/web/", http.StatusFound)

		return
	case aerr.HasTag(err, aerr.AuthenticationError):
		logger.Info().Str(common.LogKeyUserName, username).
			Str(common.LogKeyAuthResult, common.LogAuthResultFailed).
			Str(common.LogKeyAuthFailReason, aerr.GetDetails(err)).
			Msgf("web.Auth: user authentication failed user_name=%s error=%q", username, err)

	default:
		logger.Error().Err(err).Msgf("web.Auth: internal error user_name=%s error=%q", username, err)
		srvsupport.WriteError(w, r, err)
	}

	a.renderer.WriteSimplePage(w, nt.LoginPage{})
}

func (a authPages) logout(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	sess := session.GetSession(r)
	user := common.ContextUser(ctx)

	logger.Info().Str(common.LogKeyUserName, user).Msgf("web: logout user")

	sess.Flush()
	_ = sess.Destroy(w, r)
	_, _ = sess.RegenerateID(w, r)

	a.renderer.WriteSimplePage(w, nt.LogoutPage{})
}
