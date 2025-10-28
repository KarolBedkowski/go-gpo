package api

// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type deviceResource struct {
	deviceSrv *service.Device
}

func (d deviceResource) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(AuthenticatedOnly)

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.json`, internal.Wrap(d.list))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Post(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(d.update))

	return r
}

type updateDeviceReq struct {
	Caption string `json:"caption"`
	Type    string `json:"type"`
}

func (u updateDeviceReq) validate() error {
	if !slices.Contains(model.ValidDevTypes, u.Type) {
		return fmt.Errorf("invalid device type %q", u.Type) //nolint:err113
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
		logger.Debug().Err(err).Msg("error decoding json payload")
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err := udd.validate(); err != nil {
		logger.Debug().Msgf("unknown device: %q", deviceid)
		internal.WriteError(w, r, http.StatusBadRequest, err)

		return
	}

	err := d.deviceSrv.UpdateDevice(ctx, user, deviceid, udd.Caption, udd.Type)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		internal.WriteError(w, r, http.StatusBadRequest, nil)
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Debug().Msgf("unknown device: %q", deviceid)
		internal.WriteError(w, r, http.StatusBadRequest, nil)
	default:
		logger.Warn().Err(err).Msg("update device error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)
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
		internal.WriteError(w, r, http.StatusBadRequest, nil)
	default:
		logger.Warn().Err(err).Msg("update device error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)
	}
}
