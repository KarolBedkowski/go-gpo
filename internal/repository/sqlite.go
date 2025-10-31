package repository

import "github.com/samber/do/v2"

//
// sqlite.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type sqliteRepository struct{}

func NewSqliteRepository() Repository {
	return &sqliteRepository{}
}

func NewSqliteRepositoryI(i do.Injector) (Repository, error) {
	return &sqliteRepository{}, nil
}
