package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/exiguus/wdns/internal/api"
)

func TestHandlerQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()

	mux.HandleFunc("/query", func(writer http.ResponseWriter, req *http.Request) {
		var payload api.RequestPayload
		_ = json.NewDecoder(req.Body).Decode(&payload)
		resp := api.ResponsePayload{
			Status:    http.StatusOK,
			Success:   true,
			Timestamp: time.Now().Format(time.RFC3339),
			Request:   payload,
			Command:   "",
			Answer:    "93.184.216.34",
			Error:     "",
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(resp.Status)
		_ = json.NewEncoder(writer).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	reqBody := api.RequestPayload{
		Nameserver: "1.1.1.1",
		Name:       "example.com",
		Type:       "A",
		Short:      false,
		DNSSEC:     false,
		Transport:  "",
		AsJSON:     false,
	}
	b, _ := json.Marshal(reqBody)

	res, err := http.Post(srv.URL+"/query", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer res.Body.Close()

	var got api.ResponsePayload
	if decErr := json.NewDecoder(res.Body).Decode(&got); decErr != nil {
		t.Fatalf("decode failed: %v", decErr)
	}
	if !got.Success || got.Error != "" {
		t.Fatalf("unexpected response: %+v", got)
	}
}
