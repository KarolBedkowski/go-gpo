package service

//
// mod_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

func prepareTests(ctx context.Context, t *testing.T) *do.RootScope {
	t.Helper()

	i := do.New(Package, db.Package, repository.Package)

	db := do.MustInvoke[*db.Database](i)
	if err := db.Connect(ctx, "sqlite3", ":memory:"); err != nil {
		t.Fatalf("connect to db error: %#+v", err)
	}

	if err := db.Migrate(ctx, "sqlite3"); err != nil {
		t.Fatalf("prepare db error: %#+v", err)
	}

	return i
}

func prepareTestUser(ctx context.Context, t *testing.T, i do.Injector, name string) int64 {
	newuser, _ := model.NewNewUser(name, name+"123", name+"@example.com", "test user "+name)

	usersSrv := do.MustInvoke[*Users](i)
	uid, err := usersSrv.AddUser(ctx, &newuser)
	if err != nil {
		t.Fatalf("create test user failed: %#+v", err)
	}

	return uid
}
