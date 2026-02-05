package resolver_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/exiguus/wdns/internal/api"
	"github.com/exiguus/wdns/internal/resolver"
	"github.com/exiguus/wdns/internal/testutil"
)

func TestBuildKdigArgs_IncludesJSONFlag(t *testing.T) {
	req := api.RequestPayload{
		Nameserver: "ns1.example",
		Name:       "example.com",
		Type:       "AAAA",
		Transport:  "tls",
		Short:      false,
		DNSSEC:     false,
		AsJSON:     true,
	}

	args := resolver.BuildKdigArgsForTest(req)
	found := false
	for _, a := range args {
		if a == "+json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected +json in args, got: %v", args)
	}
}

func TestBuildKdigCommand_IncludesJSONFlag(t *testing.T) {
	req := api.RequestPayload{
		Nameserver: "ns1.example",
		Name:       "example.com",
		Type:       "AAAA",
		Transport:  "tls",
		Short:      false,
		DNSSEC:     false,
		AsJSON:     true,
	}

	cmd := resolver.BuildKdigCommandForTest(req)
	if !strings.Contains(cmd, "+json") {
		t.Fatalf("expected +json in command string, got: %s", cmd)
	}

	// ensure without AsJSON the flag is not present
	req.AsJSON = false
	cmd2 := resolver.BuildKdigCommandForTest(req)
	if strings.Contains(cmd2, "+json") {
		t.Fatalf("did not expect +json in command string, got: %s", cmd2)
	}
}

// Integration-ish test that runs the runner against a local UDP responder.
func TestRunnerLocalUDP(t *testing.T) {
	addr, stop := testutil.StartLocalDNSServer(t)
	defer stop()

	runner := resolver.NewRunner(2*time.Second, 1024)
	req := api.RequestPayload{
		Nameserver: addr,
		Name:       "example.com",
		Type:       "A",
		Transport:  "udp",
		Short:      true,
		DNSSEC:     false,
		AsJSON:     false,
	}

	out, cmd, err := runner.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outStr := string(out)
	if outStr == "" || cmd == "" {
		t.Fatalf("unexpected empty result: %q (cmd: %q)", outStr, cmd)
	}
}
