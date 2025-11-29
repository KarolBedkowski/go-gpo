package service

//
// maintenance.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type MaintenanceSrv struct {
	db        *db.Database
	maintRepo repository.MaintenanceRepository
}

func NewMaintenanceSrv(i do.Injector) (*MaintenanceSrv, error) {
	return &MaintenanceSrv{
		db:        do.MustInvoke[*db.Database](i),
		maintRepo: do.MustInvoke[repository.MaintenanceRepository](i),
	}, nil
}

func (m *MaintenanceSrv) MaintainDatabase(ctx context.Context) error {
	_, err := db.InConnectionR(ctx, m.db, func(ctx context.Context) (any, error) {
		return nil, m.maintRepo.Maintenance(ctx)
	})
	if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	return nil
}
