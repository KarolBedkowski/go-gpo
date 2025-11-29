package server

//
// config.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
)

type Configuration struct {
	Listen        string
	WebRoot       string
	DebugFlags    config.DebugFlags
	EnableMetrics bool
	TLSKey        string
	TLSCert       string
	CookieSecure  bool
}

func (c *Configuration) Validate() error {
	if c.Listen == "" {
		return aerr.ErrValidation.WithUserMsg("listen address can't be empty")
	}

	if (c.TLSKey != "") != (c.TLSCert != "") {
		return aerr.ErrValidation.WithUserMsg("both tls key and cert must be defined")
	}

	return nil
}

func (c *Configuration) tlsEnabled() bool {
	return c.TLSKey != ""
}

func (c *Configuration) useSecureCookie() bool {
	return c.TLSKey != "" || c.CookieSecure
}
