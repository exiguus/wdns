package handler

import (
	"net"
	"net/http"
	"strings"
)

// ipInNets reports whether ip is contained in any of the nets.
func ipInNets(ip net.IP, nets []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP extracts the client IP address from the request considering a list
// of trusted proxies. If trusted is nil or empty, it falls back to req.RemoteAddr.
// The extraction order is: `X-Forwarded-For` (left-most non-proxy), `X-Real-IP`,
// then the connection remote address.
func ClientIP(req *http.Request, trusted []*net.IPNet) string {
	// Helper to parse host:port
	hostFromRemote := func() string {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			return req.RemoteAddr
		}
		return host
	}

	// If no trusted proxies configured, return the direct remote host.
	if len(trusted) == 0 {
		return hostFromRemote()
	}

	// X-Forwarded-For: left-most is original client
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for _, p := range parts {
			ipStr := strings.TrimSpace(p)
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if !ipInNets(ip, trusted) {
				return ipStr
			}
		}
	}

	// X-Real-IP
	if xr := strings.TrimSpace(req.Header.Get("X-Real-IP")); xr != "" {
		if ip := net.ParseIP(xr); ip != nil && !ipInNets(ip, trusted) {
			return xr
		}
	}

	// Fallback to remote address
	return hostFromRemote()
}
