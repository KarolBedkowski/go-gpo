package web

//
// devices.go
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
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type devicePages struct {
	deviceSrv *service.DevicesSrv
	renderer  *nt.Renderer
}

func newDevicePages(i do.Injector) (devicePages, error) {
	return devicePages{
		deviceSrv: do.MustInvoke[*service.DevicesSrv](i),
		renderer:  do.MustInvoke[*nt.Renderer](i),
	}, nil
}

func (d devicePages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, srvsupport.WrapNamed(d.list, "web_device_list"))
	r.Get(`/{devicename:[\w.-]+}/delete`, srvsupport.WrapNamed(d.deleteGet, "web_device_del"))
	r.Post(`/{devicename:[\w.-]+}/delete`, srvsupport.WrapNamed(d.deletePost, "web_device_del_post"))

	return r
}

func (d devicePages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)

	devices, err := d.deviceSrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: user})
	if err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Devices: list user_name=%s devices error=%q", user, err)

		return
	}

	d.renderer.WritePage(w, &nt.DevicesPage{Devices: devices})
}

func (d devicePages) deleteGet(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	devicename := chi.URLParam(r, "devicename")
	if devicename == "" {
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	d.renderer.WritePage(w, &nt.DeviceDeletePage{DeviceName: devicename})
}

func (d devicePages) deletePost(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	devicename := chi.URLParam(r, "devicename")
	if devicename == "" {
		srvsupport.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	cmd := command.DeleteDeviceCmd{
		UserName:   common.ContextUser(ctx),
		DeviceName: devicename,
	}

	err := d.deviceSrv.DeleteDevice(ctx, &cmd)
	if err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Object("cmd", &cmd).
			Msgf("web.Devices: delete device_name=%s for user_name=%s error=%q", devicename, cmd.UserName, err)

		return
	}

	d.list(ctx, w, r, logger)
}
