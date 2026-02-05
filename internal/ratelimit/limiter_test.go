package ratelimit_test

import (
	"testing"
	"time"

	"github.com/exiguus/wdns/internal/ratelimit"
)

func TestAllow(t *testing.T) {
	// Use a higher refill rate and shorter sleep so the test runs faster
	mgr := ratelimit.NewManager(10.0, 1)
	remote := "127.0.0.1:12345"
	if !mgr.Allow(remote) {
		t.Fatalf("first request should be allowed")
	}
	if mgr.Allow(remote) {
		t.Fatalf("second immediate request should be denied")
	}
	// with rps=10 and burst=1, a single token should be available after ~100ms
	time.Sleep(120 * time.Millisecond)
	if !mgr.Allow(remote) {
		t.Fatalf("request after short sleep should be allowed")
	}
}
