package config

//
// server.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
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
		return ConfigurationError("listen address can't be empty")
	}

	if (c.TLSKey != "") != (c.TLSCert != "") {
		return ConfigurationError("both tls key and cert must be defined")
	}

	return nil
}

func (c *ListenConf) TLSEnabled() bool {
	return c.TLSKey != "" && c.TLSCert != ""
}

func (c *ListenConf) UseSecureCookie() bool {
	return (c.TLSKey != "" && c.TLSCert != "") || c.CookieSecure
}

func (c *ListenConf) MarshalZerologObject(event *zerolog.Event) {
	event.Str("address", c.Address).
		Str("webroot", c.WebRoot).
		Str("tls_key", c.TLSKey).
		Str("tls_cert", c.TLSCert).
		Bool("cookie_secure", c.CookieSecure)
}

//-------------------------------------------------------------

// ServerConf configure all web/api/mgmt servers.
type ServerConf struct {
	MainServer ListenConf
	MgmtServer ListenConf

	DebugFlags     DebugFlags
	EnableMetrics  bool
	MgmtAccessList string

	SetSecurityHeaders bool
	SessionStore       string

	AuthMethod      string
	ProxyUserHeader string
	ProxyAccessList string

	mgmtAccessList  *AccessList
	proxyAccessList *AccessList
}

func (c *ServerConf) Validate() error { //nolint:cyclop
	var errs error

	if err := c.MainServer.Validate(); err != nil {
		errs = errors.Join(errs, newConfigurationError("invalid server configuration: %s", err))
	}

	if c.MgmtServer.Address != "" {
		if err := c.MgmtServer.Validate(); err != nil {
			errs = errors.Join(errs, newConfigurationError("invalid mgmt configuration: %s", err))
		}
	}

	if c.MgmtAccessList != "" {
		al, err := NewAccessList(c.MgmtAccessList)
		if err != nil {
			errs = errors.Join(newConfigurationError("invalid mgmt access list: %s", err))
		}

		c.mgmtAccessList = al
	}

	switch c.SessionStore {
	case "":
		c.SessionStore = "db"
	case "db", "memory":
		// ok
	default:
		errs = errors.Join(errs, newConfigurationError("invalid session store parameter %q", c.SessionStore))
	}

	if c.ProxyAccessList != "" {
		al, err := NewAccessList(c.ProxyAccessList)
		if err != nil {
			errs = errors.Join(errs, newConfigurationError("invalid proxy access list: %s", err))
		}

		c.proxyAccessList = al
	}

	if err := c.validateAuth(); err != nil {
		errs = errors.Join(errs, err)
	}

	return errs
}

func (c *ServerConf) SeparateMgmtEnabled() bool {
	return c.MgmtServer.Address != "" && c.MgmtServer.Address != c.MainServer.Address
}

func (c *ServerConf) MgmtEnabledOnMainServer() bool {
	return c.MgmtServer.Address != "" && c.MgmtServer.Address == c.MainServer.Address
}

func (c *ServerConf) MarshalZerologObject(event *zerolog.Event) {
	event.Bool("metrics_enabled", c.EnableMetrics).
		Object("mgmt_acl", c.mgmtAccessList).
		Object("proxy_list", c.proxyAccessList).
		Str("auth_method", c.AuthMethod).
		Str("proxy_user_header", c.ProxyUserHeader).
		Bool("sec_headers", c.SetSecurityHeaders).
		Str("session_store", c.SessionStore).
		Object("main_server", &c.MainServer).
		Object("mgmt_server", &c.MgmtServer)
}

//-------------------------------------------------------------

// AuthMgmtRequest check request remote address is it allowed to access
// to debug data and sensitive information.
// Return:
//   - bool - is access allowed
//   - bool - is access to sensitive data allowed.
//
// Used for /debug (also traces and events) and /vars endpoint.
func (c *ServerConf) AuthMgmtRequest(req *http.Request) (bool, bool) {
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
		// always allow loobback
		return true, true
	case c.mgmtAccessList != nil:
		return c.mgmtAccessList.HasAccess(ip), true
	default:
		return ip.IsPrivate(), false
	}
}

func (c *ServerConf) AuthProxyRequest(remoteAddr string) bool {
	if c.proxyAccessList == nil || remoteAddr == "" {
		return false
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)

	return c.proxyAccessList.HasAccess(ip)
}

func (c *ServerConf) validateAuth() error {
	switch c.AuthMethod {
	case "":
		c.AuthMethod = "basic"
	case "basic":
		// no other options
	case "proxy":
		if c.ProxyUserHeader == "" {
			return ConfigurationError("missing proxy user header")
		}

		if c.proxyAccessList == nil || c.proxyAccessList.Len() == 0 {
			return ConfigurationError("missing proxy list")
		}
	}

	return nil
}

//-------------------------------------------------------------

type AccessList struct {
	AllowedIPs  []net.IP
	AllowedNets []*net.IPNet
}

func NewAccessList(accesslist string) (*AccessList, error) {
	var (
		ips  []net.IP
		nets []*net.IPNet
	)

	for entry := range strings.SplitSeq(accesslist, ",") {
		entry = strings.TrimSpace(entry)

		if strings.Contains(entry, "/") {
			_, n, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, newConfigurationError("invalid entry %q in access list: %s", entry, err)
			}

			nets = append(nets, n)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, newConfigurationError("invalid entry %q in access list", entry)
			}

			ips = append(ips, ip)
		}
	}

	return &AccessList{
		AllowedIPs:  ips,
		AllowedNets: nets,
	}, nil
}

func (a *AccessList) HasAccess(ip net.IP) bool {
	for _, i := range a.AllowedIPs {
		if i.Equal(ip) {
			return true
		}
	}

	for _, n := range a.AllowedNets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func (a *AccessList) Len() int {
	return len(a.AllowedNets) + len(a.AllowedIPs)
}

func (a *AccessList) MarshalZerologObject(event *zerolog.Event) {
	if a != nil {
		event.Interface("allowed_ips", a.AllowedIPs).
			Interface("allowed_nets", a.AllowedNets)
	}
}
