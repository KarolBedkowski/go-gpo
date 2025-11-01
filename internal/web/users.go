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

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type userPages struct {
	usersSrv *service.Users
	template templates
}

func newUserPages(i do.Injector) (userPages, error) {
	return userPages{
		usersSrv: do.MustInvoke[*service.Users](i),
		template: do.MustInvoke[templates](i),
	}, nil
}

func (u userPages) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get(`/password`, internal.Wrap(u.changePassword))
	r.Post(`/password`, internal.Wrap(u.changePassword))

	return r
}

func (u userPages) changePassword(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	data := struct {
		Msg string
	}{
		Msg: "",
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			logger.Info().Err(err).Msg("parse form error")
			internal.WriteError(w, r, http.StatusBadRequest, nil)

			return
		}

		data.Msg = u.doChangePassword(ctx, r, logger)
	}

	if err := u.template.executeTemplate(w, "users_change_password.tmpl", &data); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)
	}
}

func (u userPages) doChangePassword(ctx context.Context, r *http.Request, logger *zerolog.Logger) string {
	cpass, npass, msg := u.getChangePasswordParams(r)
	if msg != "" {
		return "Error: " + msg
	}

	username := internal.ContextUser(ctx)

	user, err := u.usersSrv.LoginUser(ctx, username, cpass)
	if err != nil {
		logger.Info().Err(err).Msg("check current user password for password change failed")

		return "Error: invalid current password"
	}

	user.Password = npass

	if err := u.usersSrv.ChangePassword(ctx, user); err != nil {
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
