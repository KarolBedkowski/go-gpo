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
}

func newUserPages(i do.Injector) (userPages, error) {
	return userPages{
		usersSrv: do.MustInvoke[*service.UsersSrv](i),
		renderer: do.MustInvoke[*nt.Renderer](i),
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
	u.renderer.WritePage(w, &nt.UserPage{})
}

func (u userPages) changePassword(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	var msg string

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			logger.Info().Err(err).Msg("parse form error")
			srvsupport.WriteError(w, r, http.StatusBadRequest, "")

			return
		}

		msg = u.doChangePassword(ctx, r, logger)
	}

	u.renderer.WritePage(w, &nt.UserChangePassPage{Msg: msg})
}

func (u userPages) doChangePassword(ctx context.Context, r *http.Request, logger *zerolog.Logger) string {
	cpass, npass, msg := u.getChangePasswordParams(r)
	if msg != "" {
		return "Error: " + msg
	}

	username := common.ContextUser(ctx)
	up := command.ChangeUserPasswordCmd{
		UserName: username, Password: npass, CurrentPassword: cpass, CheckCurrentPass: true,
	}

	err := u.usersSrv.ChangePassword(ctx, &up)
	if errors.Is(err, command.ErrChangePasswordOldNotMatch) {
		return "Error: invalid current password"
	} else if err != nil {
		logger.Info().Err(err).Str("user_name", username).Msg("change user password failed")

		return "Error: change password failed"
	}

	return "Password changed"
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
