// Package handler provides the HTTP handlers for the wdns service.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/exiguus/wdns/internal/api"
	"github.com/exiguus/wdns/internal/ratelimit"
	"github.com/exiguus/wdns/internal/resolver"
)

// Register registers the /query HTTP handler on the provided mux using the given
// resolver runner, optional rate limiter and optional list of trusted proxies.
// Passing a nil limiter disables rate limiting. Passing nil for trustedProxies
// means header-based client extraction is disabled and req.RemoteAddr will be
// used for rate limiting.
func Register(
	mux *http.ServeMux,
	resolverRunner *resolver.Runner,
	limiter *ratelimit.Manager,
	trustedProxies []*net.IPNet,
	logger *slog.Logger,
) {
	mux.HandleFunc("/query", makeQueryHandler(resolverRunner, limiter, trustedProxies, logger))
	// Healthcheck endpoint for readiness/liveness probes
	mux.HandleFunc("/healthz", makeHealthHandler(logger))
	mux.HandleFunc("/health", makeHealthHandler(logger))
}

func emptyRequestPayload() api.RequestPayload {
	return api.RequestPayload{
		Nameserver: "",
		Name:       "",
		Type:       "A",
		Transport:  "udp",
		DNSSEC:     false,
		Short:      false,
		AsJSON:     false,
	}
}

func writeErrorResponse(writer http.ResponseWriter, status int, req api.RequestPayload, msg string) {
	resp := api.ResponsePayload{
		Status:    status,
		Success:   false,
		Timestamp: time.Now().Format(time.RFC3339),
		Request:   req,
		Command:   "",
		Answer:    nil,
		Error:     msg,
	}
	writeJSON(writer, resp)
}

func handleRateLimit(
	writer http.ResponseWriter,
	req *http.Request,
	limiter *ratelimit.Manager,
	trusted []*net.IPNet,
) bool {
	if limiter == nil {
		return true
	}
	clientIP := req.RemoteAddr
	if len(trusted) > 0 {
		clientIP = ClientIP(req, trusted)
	} else {
		// ensure we pass only host portion
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			clientIP = host
		}
	}
	if !limiter.Allow(clientIP) {
		writer.Header().Set("Retry-After", "1")
		writeErrorResponse(writer, http.StatusTooManyRequests, emptyRequestPayload(), "rate limit exceeded")
		return false
	}
	return true
}

func decodeRequestPayload(writer http.ResponseWriter, req *http.Request) (api.RequestPayload, bool) {
	var payload api.RequestPayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeErrorResponse(writer, http.StatusBadRequest, emptyRequestPayload(), err.Error())
		return emptyRequestPayload(), false
	}
	return payload, true
}

func validatePayload(writer http.ResponseWriter, payload api.RequestPayload) bool {
	if ok, status, msg := api.Validate(payload); !ok {
		writeErrorResponse(writer, status, payload, msg)
		return false
	}
	return true
}

func makeQueryHandler(
	resolverRunner *resolver.Runner,
	limiter *ratelimit.Manager,
	trusted []*net.IPNet,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		if !handleRateLimit(writer, req, limiter, trusted) {
			return
		}

		// log incoming request for visibility
		logger.InfoContext(req.Context(), "http request",
			"method", req.Method,
			"remote", req.RemoteAddr,
			"path", req.URL.Path,
		)

		if req.Method != http.MethodPost {
			writeErrorResponse(writer, http.StatusMethodNotAllowed, emptyRequestPayload(), "Method not allowed")
			return
		}

		payload, ok := decodeRequestPayload(writer, req)
		if !ok {
			return
		}

		if !validatePayload(writer, payload) {
			return
		}

		// log the query payload for every request
		clientIP := ClientIP(req, trusted)
		logger.InfoContext(req.Context(), "query payload",
			"nameserver", payload.Nameserver,
			"name", payload.Name,
			"type", payload.Type,
			"transport", payload.Transport,
			"dnssec", payload.DNSSEC,
			"short", payload.Short,
			"json", payload.AsJSON,
			"client", clientIP,
		)

		// Use request context and a safety timeout
		ctx := req.Context()
		ctx, cancel := context.WithTimeout(ctx, resolverRunner.Timeout+1*time.Second)
		defer cancel()

		out, cmdDesc, runErr := resolverRunner.Run(ctx, payload)

		// log empty responses (no output) for visibility
		if len(out) == 0 {
			client := ClientIP(req, trusted)
			logger.InfoContext(req.Context(), "empty resolver response",
				"nameserver", payload.Nameserver,
				"name", payload.Name,
				"type", payload.Type,
				"transport", payload.Transport,
				"dnssec", payload.DNSSEC,
				"client", client,
				"error", runErr,
			)
		}

		resp := api.ResponsePayload{
			Status:    http.StatusOK,
			Success:   runErr == nil,
			Timestamp: time.Now().Format(time.RFC3339),
			Request:   payload,
			Command:   cmdDesc,
			Answer:    nil,
			Error:     "",
		}

		if runErr != nil {
			resp.Status = http.StatusInternalServerError
			resp.Error = runErr.Error()
		}

		if payload.AsJSON && len(out) > 0 {
			var parsed interface{}
			if unmarshalErr := json.Unmarshal(out, &parsed); unmarshalErr == nil {
				resp.Answer = parsed
			} else {
				resp.Answer = string(out)
			}
		} else {
			resp.Answer = string(out)
		}

		writeJSON(writer, resp)
	}
}

func writeJSON(w http.ResponseWriter, resp api.ResponsePayload) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	_ = json.NewEncoder(w).Encode(resp)
}

// makeHealthHandler returns a simple healthcheck handler that responds 200 OK
// with a small JSON body. Useful for liveness/readiness probes.
func makeHealthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		// Log the health check for visibility
		if logger != nil {
			logger.InfoContext(req.Context(), "http request",
				"method", req.Method,
				"remote", req.RemoteAddr,
				"path", req.URL.Path,
			)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(map[string]string{"status": "ok"})
	}
}
