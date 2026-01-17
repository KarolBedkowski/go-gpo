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
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func (c *ServerConf) Validate() error {
	if err := c.MainServer.Validate(); err != nil {
		return fmt.Errorf("validate main server configuration failed: %w", err)
	}

	if c.MgmtServer.Address != "" {
		if err := c.MgmtServer.Validate(); err != nil {
			return fmt.Errorf("validate mgmt server configuration failed: %w", err)
		}
	}

	if c.MgmtAccessList != "" {
		al, err := NewAccessList(c.MgmtAccessList)
		if err != nil {
			return fmt.Errorf("validate mgmt access list failed: %w", err)
		}

		c.mgmtAccessList = al

		log.Logger.Debug().Object("debugAccessList", al).Msg("debug access list configured")
	}

	switch c.SessionStore {
	case "":
		c.SessionStore = "db"
	case "db", "memory":
		// ok
	default:
		return aerr.ErrValidation.WithUserMsg("invalid session store parameter")
	}

	if err := c.validateAuth(); err != nil {
		return err
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
	if remoteAddr == "" {
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
			return aerr.ErrValidation.WithUserMsg("missing proxy user header")
		}

		if c.ProxyAccessList == "" {
			return aerr.ErrValidation.WithUserMsg("missing proxy access list")
		}

		al, err := NewAccessList(c.ProxyAccessList)
		if err != nil {
			return fmt.Errorf("validate proxy access list failed: %w", err)
		}

		c.proxyAccessList = al

		log.Logger.Debug().Object("proxyAccessList", al).Msg("proxy access list configured")
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
				return nil, aerr.ErrValidation.WithUserMsg(
					"invalid entry in access list: entry=%q error=%q", entry, err)
			}

			nets = append(nets, n)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, aerr.ErrValidation.WithUserMsg("invalid entry in access list: entry=%q", entry)
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

func (a *AccessList) MarshalZerologObject(event *zerolog.Event) {
	event.Interface("allowed_ips", a.AllowedIPs).
		Interface("allowed_nets", a.AllowedNets)
}
