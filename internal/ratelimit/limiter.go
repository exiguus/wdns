package ratelimit

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Manager holds per-client rate limiters.
type Manager struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      float64
	burst    int
}

// NewManager creates a Manager with the given rps and burst settings.
func NewManager(rps float64, burst int) *Manager {
	return &Manager{mu: sync.Mutex{}, limiters: make(map[string]*rate.Limiter), rps: rps, burst: burst}
}

// getIP extracts an IP address (no port) from host:port or returns the input if plain IP.
func getIP(hostport string) string {
	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return host
	}
	return hostport
}

// Allow reports whether the given remote (host:port or IP) is allowed.
func (m *Manager) Allow(remote string) bool {
	ip := getIP(remote)
	m.mu.Lock()
	limiter, ok := m.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(m.rps), m.burst)
		m.limiters[ip] = limiter
	}
	m.mu.Unlock()

	return limiter.Allow()
}

// Cleanup removes limiters that have no tokens (not perfect but keeps map small over time).
func (m *Manager) Cleanup(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			for ip, limiter := range m.limiters {
				// if the limiter permits a full burst, treat it as idle
				if limiter.AllowN(time.Now(), m.burst) {
					delete(m.limiters, ip)
				}
			}
			m.mu.Unlock()
		case <-stop:
			return
		}
	}
}
