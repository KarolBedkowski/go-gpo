package web

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type userPages struct {
	usersSrv *service.UsersSrv
	renderer *nt.Renderer
	webroot  string
}

func newUserPages(i do.Injector) (userPages, error) {
	return userPages{
		usersSrv: do.MustInvoke[*service.UsersSrv](i),
		renderer: do.MustInvoke[*nt.Renderer](i),
		webroot:  do.MustInvokeNamed[string](i, "server.webroot"),
	}, nil
}

func (u userPages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, srvsupport.WrapNamed(u.userPage, "web_user_index"))
	r.Get(`/password`, srvsupport.WrapNamed(u.changePassword, "web_user_pass"))
	r.Post(`/password`, srvsupport.WrapNamed(u.changePassword, "web_user_pass_post"))

	return r
}

func (u userPages) userPage(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	u.renderer.WritePage(w, r, &nt.UserPage{})
}

func (u userPages) changePassword(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	if r.Method == http.MethodPost {
		const maxBody = 1024 * 1024 // 1k

		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		if err := r.ParseForm(); err != nil {
			logger.Info().Err(err).Msgf("web.User: bad request - parse form error=%q", err)
			u.renderer.WriteBadRequestError(w, r, "Invalid form data.")

			return
		}

		switch msg, ok := u.doChangePassword(ctx, r, logger); {
		case ok:
			nt.AddFlash(r, "success", "Password changed.")
			http.Redirect(w, r, u.webroot+"/web/user/", http.StatusFound)

			return
		case msg != "":
			nt.AddFlash(r, "error", msg)
		}
	}

	u.renderer.WritePage(w, r, &nt.UserChangePassPage{})
}

func (u userPages) doChangePassword(ctx context.Context, r *http.Request, logger *zerolog.Logger) (string, bool) {
	cpass, npass, msg := u.getChangePasswordParams(r)
	if msg != "" {
		return "Error: " + msg, false
	}

	username := common.ContextUser(ctx)
	up := command.ChangeUserPasswordCmd{
		UserName: username, Password: npass, CurrentPassword: cpass, CheckCurrentPass: true,
	}

	err := u.usersSrv.ChangePassword(ctx, &up)
	if errors.Is(err, command.ErrChangePasswordOldNotMatch) {
		return "Error: invalid current password", false
	} else if err != nil {
		logger.Info().Err(err).Str("user_name", username).
			Msgf("web.User: change user_name=%s password error=%q", username, err)

		return "Error: change password failed", false
	}

	return "Password changed", true
}

func (userPages) getChangePasswordParams(r *http.Request) (string, string, string) {
	currentPass := r.FormValue("cpass")
	newpass1 := r.FormValue("npass1")
	newpass2 := r.FormValue("npass2")

	if newpass1 != newpass2 {
		return "", "", "new passwords do not match"
	}

	if currentPass == "" {
		return "", "", "current password can't be empty"
	}

	if newpass1 == "" {
		return "", "", "new password can't be empty"
	}

	return currentPass, newpass1, ""
}
