package api_test

import (
	"testing"

	"github.com/exiguus/wdns/internal/api"
)

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  api.RequestPayload
		ok   bool
	}{
		{
			"empty nameserver",
			api.RequestPayload{
				Nameserver: "",
				Name:       "example.com",
				Type:       "A",
				Short:      false,
				DNSSEC:     false,
				Transport:  "",
				AsJSON:     false,
			},
			false,
		},
		{
			"empty name",
			api.RequestPayload{
				Nameserver: "1.1.1.1",
				Name:       "",
				Type:       "A",
				Short:      false,
				DNSSEC:     false,
				Transport:  "",
				AsJSON:     false,
			},
			false,
		},
		{
			"bad type",
			api.RequestPayload{
				Nameserver: "1.1.1.1",
				Name:       "example.com",
				Type:       "TXT",
				Short:      false,
				DNSSEC:     false,
				Transport:  "",
				AsJSON:     false,
			},
			false,
		},
		{
			"bad transport",
			api.RequestPayload{
				Nameserver: "1.1.1.1",
				Name:       "example.com",
				Type:       "A",
				Transport:  "bad",
				Short:      false,
				DNSSEC:     false,
				AsJSON:     false,
			},
			false,
		},
		{
			"ok",
			api.RequestPayload{
				Nameserver: "1.1.1.1",
				Name:       "example.com",
				Type:       "A",
				Transport:  "",
				Short:      false,
				DNSSEC:     false,
				AsJSON:     false,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok, _, _ := api.Validate(tt.req)
			if ok != tt.ok {
				t.Fatalf("expected %v, got %v", tt.ok, ok)
			}
		})
	}
}
