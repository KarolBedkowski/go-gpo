package server

//
// config.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"net"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
)

type Configuration struct {
	Listen        string
	WebRoot       string
	TLSKey        string
	TLSCert       string
	DebugFlags    config.DebugFlags
	EnableMetrics bool
	CookieSecure  bool

	MgmtListen  string
	MgmtWebRoot string
	MgmtTLSKey  string
	MgmtTLSCert string
}

func (c *Configuration) Validate() error {
	if c.Listen == "" {
		return aerr.ErrValidation.WithUserMsg("listen address can't be empty")
	}

	if (c.TLSKey != "") != (c.TLSCert != "") {
		return aerr.ErrValidation.WithUserMsg("both tls key and cert must be defined")
	}

	if c.MgmtListen != "" {
		if (c.MgmtTLSKey != "") != (c.MgmtTLSCert != "") {
			return aerr.ErrValidation.WithUserMsg("both tls key and cert must be defined")
		}
	}

	return nil
}

func (c *Configuration) SeparateMgmtEnabled() bool {
	return c.MgmtListen != "" && c.MgmtListen != c.Listen
}

func (c *Configuration) mgmtEnabledOnMainServer() bool {
	return c.MgmtListen != "" && c.MgmtListen == c.Listen
}

func (c *Configuration) tlsEnabled() bool {
	return c.TLSKey != ""
}

func (c *Configuration) mgmtTLSEnabled() bool {
	return c.MgmtTLSKey != ""
}

func (c *Configuration) useSecureCookie() bool {
	return c.TLSKey != "" || c.CookieSecure
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
