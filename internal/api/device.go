// auth.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/hlog"
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
		r.Use(checkUserMiddleware)
	}

	r.Get("/{user:[0-9a-z._-]+}.json", d.list)
	r.Post("/{user:[0-9a-z_.-]+}/{deviceid:[0-9a-z_.-]+}.json", d.update)

	return r
}

func (d deviceResource) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	type updateDeviceData struct {
		Caption string `json:"caption"`
		Type    string `json:"type"`
	}

	udd := updateDeviceData{}
	if err := render.DecodeJSON(r.Body, &udd); err != nil {
		logger.Info().Err(err).Msg("error decoding json payload")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	err := d.deviceSrv.UpdateDevice(ctx, user, deviceid, udd.Caption, udd.Type)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Info().Msgf("unknown device: %q", deviceid)
		w.WriteHeader(http.StatusBadRequest)
	default:
		logger.Info().Err(err).Msg("update device error")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (d deviceResource) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	devices, err := d.deviceSrv.ListDevices(ctx, user)
	switch {
	case err == nil:
		render.JSON(w, r, devices)
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		w.WriteHeader(http.StatusBadRequest)
	default:
		logger.Info().Err(err).Msg("update device error")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
