package config

//
// server.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"net"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// ListenConf configure one address on server.
type ListenConf struct {
	Address      string
	WebRoot      string
	TLSKey       string
	TLSCert      string
	CookieSecure bool
}

func (c *ListenConf) Validate() error {
	if c.Address == "" {
		return aerr.ErrValidation.WithUserMsg("listen address can't be empty")
	}

	if (c.TLSKey != "") != (c.TLSCert != "") {
		return aerr.ErrValidation.WithUserMsg("both tls key and cert must be defined")
	}

	return nil
}

func (c *ListenConf) TLSEnabled() bool {
	return c.TLSKey != ""
}

func (c *ListenConf) UseSecureCookie() bool {
	return c.TLSKey != "" || c.CookieSecure
}

//-------------------------------------------------------------

// ServerConf configure all web/api/mgmt servers.
type ServerConf struct {
	MainServer ListenConf
	MgmtServer ListenConf

	DebugFlags    DebugFlags
	EnableMetrics bool
}

func (c *ServerConf) Validate() error {
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

func (c *ServerConf) SeparateMgmtEnabled() bool {
	return c.MgmtServer.Address != "" && c.MgmtServer.Address != c.MainServer.Address
}

func (c *ServerConf) MgmtEnabledOnMainServer() bool {
	return c.MgmtServer.Address != "" && c.MgmtServer.Address == c.MainServer.Address
}

//-------------------------------------------------------------

// AuthDebugRequest check request remote address is it allowed to access
// to debug data and sensitive information.
// Return:
//   - bool - is access allowed
//   - bool - is access to sensitive data allowed.
//
// Used for /debug (also traces and events) and /vars endpoint.
func (c *ServerConf) AuthDebugRequest(req *http.Request) (bool, bool) {
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
