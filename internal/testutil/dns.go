package testutil

import (
	"net"
	"strconv"
	"testing"
	"time"
)

const (
	bufSize            = 512
	headerLen          = 12
	ttlSecs            = 60
	anCountIndex       = 6
	anCountShift       = 8
	dialTimeoutMs      = 200
	responderStartupMs = 10
)

// StartLocalDNSServer starts a minimal UDP DNS server that responds with a
// single A record for example.com. It returns the listening address
// (host:port) and a stop function to shut the server down.
func StartLocalDNSServer(t *testing.T) (string, func()) {
	t.Helper()

	var lc net.ListenConfig
	conn, listenErr := lc.ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if listenErr != nil {
		t.Fatalf("listen udp: %v", listenErr)
	}

	localAddr := conn.LocalAddr()
	udpAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		_ = conn.Close()
		t.Fatalf("unexpected address type: %T", localAddr)
	}

	srvAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(udpAddr.Port))

	stopCh := make(chan struct{})
	go runUDPResponder(conn, stopCh)

	// The responder is ready once the packet listener is created and the
	// responder goroutine is started. A brief sleep gives the goroutine time
	// to begin reading; full readiness probing via UDP requests is flaky on
	// CI, so keep the startup logic simple and deterministic.
	time.Sleep(responderStartupMs * time.Millisecond)

	return srvAddr, func() { close(stopCh); _ = conn.Close() }
}

// runUDPResponder handles incoming UDP DNS requests and writes a fixed A answer.
func runUDPResponder(conn net.PacketConn, stopCh chan struct{}) {
	buf := make([]byte, bufSize)
	// Local copies of DNS reply fragments to avoid package-level globals.
	respFlags := []byte{0x81, 0x80}
	respANCount := []byte{0x00, 0x01}
	respZeroTwoBytes := []byte{0x00, 0x00}
	respPointerToQ := []byte{0xc0, 0x0c}
	respTypeA := []byte{0x00, 0x01}
	respTTL := []byte{0x00, 0x00, 0x00, 0x3c}
	respRDLength := []byte{0x00, 0x04}
	respRData := []byte{93, 184, 216, 34}
	for {
		select {
		case <-stopCh:
			_ = conn.Close()
			return
		default:
		}
		n, remote, readErr := conn.ReadFrom(buf)
		if readErr != nil {
			continue
		}
		req := make([]byte, n)
		copy(req, buf[:n])
		if len(req) < headerLen {
			continue
		}

		idBytes := req[0:2]
		qdCount := req[4:6]

		// find end of qname
		offset := 12
		for offset < len(req) && req[offset] != 0 {
			labelLen := int(req[offset])
			offset += 1 + labelLen
		}
		if offset+5 > len(req) {
			continue
		}
		qSection := req[12 : offset+5]

		resp := make([]byte, 0, bufSize)
		resp = append(resp, idBytes...)
		resp = append(resp, respFlags...)
		resp = append(resp, qdCount...)
		resp = append(resp, respANCount...)
		// NSCOUNT (0) and ARCOUNT (0)
		resp = append(resp, respZeroTwoBytes...)
		resp = append(resp, respZeroTwoBytes...)
		resp = append(resp, qSection...)
		resp = append(resp, respPointerToQ...)
		resp = append(resp, respTypeA...)
		resp = append(resp, respTypeA...)
		resp = append(resp, respTTL...)
		resp = append(resp, respRDLength...)
		resp = append(resp, respRData...)

		_, _ = conn.WriteTo(resp, remote)
	}
}

// (no exported helpers)
