// Package config provide application configuration objects.
package config

import "fmt"

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type ConfigurationError string

func (c ConfigurationError) Error() string {
	return string(c)
}

func newConfigurationError(msg string, args ...any) ConfigurationError {
	return ConfigurationError(fmt.Sprintf(msg, args...))
}
