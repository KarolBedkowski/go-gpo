package web //nolint:dupl

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
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/queries"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type devicePages struct {
	deviceSrv *service.DevicesSrv
	template  templates
}

func newDevicePages(i do.Injector) (devicePages, error) {
	return devicePages{
		deviceSrv: do.MustInvoke[*service.DevicesSrv](i),
		template:  do.MustInvoke[templates](i),
	}, nil
}

func (d devicePages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, internal.Wrap(d.list))

	return r
}

func (d devicePages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	devices, err := d.deviceSrv.ListDevices(ctx, &queries.QueryDevices{UserName: user})
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("list devices error")

		return
	}

	data := struct {
		Devices []model.Device
	}{
		Devices: devices,
	}

	if err := d.template.executeTemplate(w, "devices.tmpl", &data); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		internal.WriteError(w, r, http.StatusInternalServerError, "")
	}
}
