package api

// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// deviceResource handle request to /api/2/devices resource.
type deviceResource struct {
	deviceSrv *service.DevicesSrv
}

func newDeviceResource(i do.Injector) (deviceResource, error) {
	return deviceResource{
		deviceSrv: do.MustInvoke[*service.DevicesSrv](i),
	}, nil
}

func (d deviceResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.json`, srvsupport.WrapNamed(d.listDevices, "api_dev_user"))
	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Post(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.json`, srvsupport.WrapNamed(d.updateDevice, "api_dev_user_put"))

	return r
}

// updateDevice device data.
func (d deviceResource) updateDevice(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)
	devicename := common.ContextDevice(ctx)

	// updateDevice device data
	var reqData struct {
		Caption string `json:"caption"`
		Type    string `json:"type"`
	}

	if err := render.DecodeJSON(r.Body, &reqData); err != nil {
		logger.Debug().Err(err).Msg("error decoding json payload")
		writeError(w, r, http.StatusBadRequest)

		return
	}

	cmd := command.UpdateDeviceCmd{
		UserName:   user,
		DeviceName: devicename,
		DeviceType: reqData.Type,
		Caption:    reqData.Caption,
	}
	if err := d.deviceSrv.UpdateDevice(ctx, &cmd); err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("updateDevice device error")

		return
	}

	w.WriteHeader(http.StatusOK)
}

// listDevices devices.
func (d deviceResource) listDevices(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)

	devices, err := d.deviceSrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: user})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get devices error")

		return
	}

	resdevices := common.Map(devices, newDeviceFromModel)

	render.Status(r, http.StatusOK)
	srvsupport.RenderJSON(w, r, resdevices)
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
		User:          d.User.Name,
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
	}
}
