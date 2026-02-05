package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// LoadTrustedProxies reads the TRUSTED_PROXIES environment variable, which is a
// comma-separated list of CIDRs (e.g. "10.0.0.0/8,192.168.0.0/16"). It returns
// a slice of parsed *net.IPNet. An empty value returns an empty slice and no error.
func LoadTrustedProxies() ([]*net.IPNet, error) {
	raw := os.Getenv("TRUSTED_PROXIES")
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	var out []*net.IPNet
	for _, p := range parts {
		cidr := strings.TrimSpace(p)
		if cidr == "" {
			continue
		}
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		out = append(out, ipnet)
	}
	return out, nil
}
