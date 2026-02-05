package handler_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/exiguus/wdns/internal/handler"
)

func TestClientIP_XForwardedFor(t *testing.T) {
	// trusted proxy covering 10.0.0.0/8
	_, trustedNet, _ := net.ParseCIDR("10.0.0.0/8")
	trusted := []*net.IPNet{trustedNet}

	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := handler.ClientIP(req, trusted)
	if ip != "203.0.113.5" {
		t.Fatalf("expected client ip 203.0.113.5, got %q", ip)
	}
}

func TestClientIP_NoTrusted(t *testing.T) {
	req := &http.Request{Header: make(http.Header)}
	req.RemoteAddr = "192.0.2.1:54321"
	ip := handler.ClientIP(req, nil)
	if ip != "192.0.2.1" {
		t.Fatalf("expected remote addr host 192.0.2.1, got %q", ip)
	}
}
