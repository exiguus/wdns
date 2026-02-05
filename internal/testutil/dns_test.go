package testutil_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/exiguus/wdns/internal/testutil"
)

const (
	testBufSize = 512
)

func TestStartLocalDNSServer(t *testing.T) {
	addr, stop := testutil.StartLocalDNSServer(t)
	defer stop()

	// send a raw query and ensure we receive an answer (ANCOUNT > 0)
	queryBytes := buildQueryBytes("example.com")
	var dialer net.Dialer
	dialer.Timeout = 2 * time.Second
	conn, err := dialer.DialContext(context.Background(), "udp", addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	_, _ = conn.Write(queryBytes)
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	resp := make([]byte, 512)
	n, err := conn.Read(resp)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if n < 12 {
		t.Fatalf("response too short")
	}
	an := int(resp[6])<<8 | int(resp[7])
	if an == 0 {
		t.Fatalf("expected an answer, got none")
	}
}

// buildQueryBytes duplicates the internal helper to avoid importing unexported symbols.
func buildQueryBytes(name string) []byte {
	// ID and flags header (recursion desired), QDCOUNT=1
	// Full 12-byte DNS header: ID, Flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=0, ARCOUNT=0
	initialHeader := []byte{0x12, 0x34, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	typeA := []byte{0x00, 0x01}
	buf := make([]byte, 0, testBufSize)
	buf = append(buf, initialHeader...)
	for _, label := range strings.Split(name, ".") {
		if label == "" {
			continue
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}
	buf = append(buf, 0x00)
	buf = append(buf, typeA...) // QTYPE A
	buf = append(buf, typeA...) // QCLASS IN
	return buf
}
