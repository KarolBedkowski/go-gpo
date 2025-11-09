package api

// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"
	"slices"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type deviceResource struct {
	deviceSrv *service.Device
}

func newDeviceResource(i do.Injector) (deviceResource, error) {
	return deviceResource{
		deviceSrv: do.MustInvoke[*service.Device](i),
	}, nil
}

func (d deviceResource) Routes() *chi.Mux {
	r := chi.NewRouter()

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
		return aerr.Newf("invalid device type %q", u.Type).WithTag(aerr.ValidationError).
			WithUserMsg("invalid device type")
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
		logger.Debug().Err(err).Str("mod", "api").Msg("error decoding json payload")
		internal.WriteError(w, r, http.StatusBadRequest, "bad request data")

		return
	}

	if err := udd.validate(); err != nil {
		logger.Debug().Err(err).Msgf("validation error")
		internal.WriteError(w, r, http.StatusBadRequest, aerr.GetUserMessageOr(err, "bad request data"))

		return
	}

	if err := d.deviceSrv.UpdateDevice(ctx, user, deviceid, udd.Caption, udd.Type); err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("update device error")

		return
	}

	w.WriteHeader(http.StatusOK)
}

// list devices.
func (d deviceResource) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	devices, err := d.deviceSrv.ListDevices(ctx, user)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get devices error")

		return
	}

	resdevices := make([]device, len(devices))
	for i, d := range devices {
		resdevices[i] = newDeviceFromModel(&d)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, resdevices)
}

type device struct {
	User          string `json:"user"`
	Name          string `json:"id"`
	DevType       string `json:"type"`
	Caption       string `json:"caption"`
	Subscriptions int    `json:"subscriptions"`
}

func newDeviceFromModel(d *model.Device) device {
	return device{
		User:          d.User,
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
	}
}
