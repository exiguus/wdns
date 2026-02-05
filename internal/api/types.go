// Package api contains request/response types and validation for the wdns HTTP API.
package api

import "net/http"

// RequestPayload defines the structure of the incoming JSON request.
//
// Fields correspond to the input accepted by the HTTP `/query` endpoint.
type RequestPayload struct {
	Nameserver string `json:"nameserver"`
	Short      bool   `json:"short"`
	DNSSEC     bool   `json:"dnssec"`
	Type       string `json:"type"`
	Transport  string `json:"transport"`
	Name       string `json:"name"`
	AsJSON     bool   `json:"json"`
}

// ResponsePayload defines the structure of the JSON responses.
//
// It contains both request echo, executed command and the answer or error.
type ResponsePayload struct {
	Status    int            `json:"status"`
	Success   bool           `json:"success"`
	Timestamp string         `json:"timestamp"`
	Request   RequestPayload `json:"request"`
	// Command contains a kdig-equivalent command that represents the query executed.
	Command string      `json:"command,omitempty"`
	Answer  interface{} `json:"answer,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Validate checks request parameters and returns (ok, httpStatus, errorMessage).
//
// It applies basic presence and allowed-value checks for incoming requests.
func Validate(req RequestPayload) (bool, int, string) {
	if req.Nameserver == "" {
		return false, http.StatusBadRequest, `"nameserver" must not be empty`
	}
	if req.Name == "" {
		return false, http.StatusBadRequest, `"name" must not be empty`
	}
	if req.Type != "AAAA" && req.Type != "A" {
		return false, http.StatusBadRequest, `"type" must be "AAAA" or "A"`
	}
	if req.Transport != "tls" && req.Transport != "https" && req.Transport != "tcp" && req.Transport != "" {
		return false, http.StatusBadRequest, `"transport" must be empty or "tcp" or "tls" or "https"`
	}
	return true, http.StatusOK, ""
}
