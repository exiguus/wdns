// Package wdns provides the application entrypoint and server startup logic.
package wdns

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/exiguus/wdns/internal/config"
	"github.com/exiguus/wdns/internal/handler"
	"github.com/exiguus/wdns/internal/ratelimit"
	"github.com/exiguus/wdns/internal/resolver"
)

const (
	defaultResolverTimeout = 5 * time.Second
	defaultMaxOutput       = 32 * 1024
	shutdownTimeout        = 10 * time.Second
	readHeaderTimeout      = 5 * time.Second
	readTimeout            = 10 * time.Second
	writeTimeout           = 10 * time.Second
	cleanupInterval        = 5 * time.Minute
)

// Run starts the HTTP server and registers the application handlers.
// It is exported so `cmd/wdns` can call into the package to produce the
// executable while allowing the core logic to be imported by other code.
func Run() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	// create logger early so we can log during startup
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// create resolver runner
	resolverRunner := resolver.NewRunner(defaultResolverTimeout, defaultMaxOutput)

	mux := http.NewServeMux()
	// initialize rate limiter
	limiter, stopCleanup := createLimiter()
	defer close(stopCleanup)
	// load trusted proxies for header-based client IP extraction
	trustedProxies, err := config.LoadTrustedProxies()
	if err != nil {
		log.Printf("warning: failed to parse TRUSTED_PROXIES: %v", err)
	}
	// pass logger to handler for request-level logging
	handler.Register(mux, resolverRunner, limiter, trustedProxies, logger)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
	}

	go func() {
		log.Printf("Server started on %s", srv.Addr)
		if serr := srv.ListenAndServe(); serr != nil && serr != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", serr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	// call cancel before possible exit to ensure resources are released
	if derr := srv.Shutdown(ctx); derr != nil {
		cancel()
		log.Printf("Server Shutdown failed:%+v", derr)
		return
	}
	cancel()
	log.Println("Server exited properly")
}

// createLimiter reads env vars and returns a configured rate limiter and a stop channel.
func createLimiter() (*ratelimit.Manager, chan struct{}) {
	rps := 10.0
	burst := 20
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			rps = parsed
		}
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			burst = parsed
		}
	}
	limiter := ratelimit.NewManager(rps, burst)
	stopCleanup := make(chan struct{})
	go limiter.Cleanup(cleanupInterval, stopCleanup)
	return limiter, stopCleanup
}
