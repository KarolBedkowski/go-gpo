package server

//
// config.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"net"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
)

type ListenConfiguration struct {
	Address      string
	WebRoot      string
	TLSKey       string
	TLSCert      string
	CookieSecure bool
}

func (c *ListenConfiguration) Validate() error {
	if c.Address == "" {
		return aerr.ErrValidation.WithUserMsg("listen address can't be empty")
	}

	if (c.TLSKey != "") != (c.TLSCert != "") {
		return aerr.ErrValidation.WithUserMsg("both tls key and cert must be defined")
	}

	return nil
}

func (c *ListenConfiguration) tlsEnabled() bool {
	return c.TLSKey != ""
}

func (c *ListenConfiguration) useSecureCookie() bool {
	return c.TLSKey != "" || c.CookieSecure
}

//-------------------------------------------------------------

type Configuration struct {
	MainServer ListenConfiguration
	MgmtServer ListenConfiguration

	DebugFlags    config.DebugFlags
	EnableMetrics bool
}

func (c *Configuration) Validate() error {
	if err := c.MainServer.Validate(); err != nil {
		return fmt.Errorf("validate main server configuration failed: %w", err)
	}

	if c.MgmtServer.Address != "" {
		if err := c.MgmtServer.Validate(); err != nil {
			return fmt.Errorf("validate mgmt server configuration failed: %w", err)
		}
	}

	return nil
}

func (c *Configuration) SeparateMgmtEnabled() bool {
	return c.MgmtServer.Address != "" && c.MgmtServer.Address != c.MainServer.Address
}

func (c *Configuration) mgmtEnabledOnMainServer() bool {
	return c.MgmtServer.Address != "" && c.MgmtServer.Address == c.MainServer.Address
}

// authDebugRequest check request remote address is it allowed to access
// to debug data and sensitive information.
// Return:
//   - bool - is access allowed
//   - bool - is access to sensitive data allowed.
//
// Used for /debug (also traces and events) and /vars endpoint.
func (c *Configuration) authDebugRequest(req *http.Request) (bool, bool) {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}

	if host == "localhost" {
		return true, true
	}

	ip := net.ParseIP(host)
	switch {
	case ip == nil:
		return false, false
	case ip.IsLoopback():
		return true, true
	default:
		return ip.IsPrivate(), false
	}
}
