package api

// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpodder/internal"
	apperrors "gitlab.com/kabes/go-gpodder/internal/errors"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type deviceResource struct {
	cfg       *Configuration
	deviceSrv *service.Device
}

func (d deviceResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !d.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
	}

	r.With(checkUserMiddleware).
		Get("/{user:[0-9a-z._-]+}.json", wrap(d.list))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Post("/{user:[0-9a-z_.-]+}/{deviceid:[0-9a-z_.-]+}.json", wrap(d.update))

	return r
}

type updateDeviceReq struct {
	Caption string `json:"caption"`
	Type    string `json:"type"`
}

func (u updateDeviceReq) validate() error {
	if !slices.Contains(model.ValidDevTypes, u.Type) {
		return apperrors.NewAppError("invalid device type %q", u.Type).WithCategory(apperrors.ValidationError)
	}

	return nil
}

// update device data.
func (d deviceResource) update(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	// update device data
	var udd updateDeviceReq

	if err := render.DecodeJSON(r.Body, &udd); err != nil {
		logger.Info().Err(err).Msg("error decoding json payload")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err := udd.validate(); err != nil {
		logger.Info().Msgf("unknown device: %q", deviceid)
		writeError(w, r, http.StatusBadRequest, err)

		return
	}

	err := d.deviceSrv.UpdateDevice(ctx, user, deviceid, udd.Caption, udd.Type)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		writeError(w, r, http.StatusBadRequest, nil)
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Info().Msgf("unknown device: %q", deviceid)
		writeError(w, r, http.StatusBadRequest, nil)
	default:
		logger.Info().Err(err).Msg("update device error")
		writeError(w, r, http.StatusInternalServerError, nil)
	}
}

// list devices.
func (d deviceResource) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	devices, err := d.deviceSrv.ListDevices(ctx, user)
	switch {
	case err == nil:
		render.Status(r, http.StatusOK)
		render.JSON(w, r, ensureList(devices))
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		writeError(w, r, http.StatusBadRequest, nil)
	default:
		logger.Info().Err(err).Msg("update device error")
		writeError(w, r, http.StatusInternalServerError, nil)
	}
}
