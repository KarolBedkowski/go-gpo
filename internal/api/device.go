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
	deviceSrv *service.Device
}

func (dr deviceResource) Routes() chi.Router {
	r := chi.NewRouter()
	// r.Use(AuthenticatedOnly)

	r.Get("/{user:[0-9a-z.-]+}.json", dr.list)
	r.Post("/{user:[0-9a-z.-]+}/{deviceid:[0-9a-z.-]+}.json", dr.update)
	return r
}

func (d *deviceResource) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")
	suser := userFromContext(ctx)
	if suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		// w.WriteHeader(http.StatusBadRequest)
		// return
	}

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
	err := render.DecodeJSON(r.Body, &udd)
	if err != nil {
		logger.Info().Err(err).Msg("error decoding json payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = d.deviceSrv.UpdateDevice(ctx, user, deviceid, udd.Caption, udd.Type)
	switch {
	case err == nil:
		w.WriteHeader(200)
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

func (d *deviceResource) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		// w.WriteHeader(http.StatusBadRequest)
		// return
	}

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
