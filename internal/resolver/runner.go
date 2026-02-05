package resolver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/exiguus/wdns/internal/api"
)

// Runner executes DNS queries by invoking the external `kdig` binary.
// It returns the raw stdout from kdig and a human-friendly command string
// useful for debugging and reproducing queries.
type Runner struct {
	Timeout   time.Duration
	MaxOutput int
	logger    *slog.Logger
}

// NewRunner creates a new Runner.
func NewRunner(timeout time.Duration, maxOutput int) *Runner {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return &Runner{Timeout: timeout, MaxOutput: maxOutput, logger: logger}
}

// Run builds and executes a corresponding kdig command for the request.
// It returns the command's stdout, the human command string, and any error.
func (r *Runner) Run(ctx context.Context, req api.RequestPayload) ([]byte, string, error) {
	cmdStr := buildKdigCommand(req)

	args := buildKdigArgs(req)
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kdig", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			err = fmt.Errorf("kdig failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		} else {
			err = fmt.Errorf("kdig failed: %w", err)
		}
		r.logger.ErrorContext(ctx, "resolver: kdig execution failed",
			slog.String("nameserver", req.Nameserver),
			slog.String("name", req.Name),
			slog.String("transport", req.Transport),
			slog.Any("err", err),
		)
		return nil, cmdStr, err
	}

	if r.MaxOutput > 0 && len(out) > r.MaxOutput {
		out = out[:r.MaxOutput]
	}

	return out, cmdStr, nil
}

// buildKdigCommand creates a human-readable kdig command string.
func buildKdigCommand(req api.RequestPayload) string {
	var builder strings.Builder
	builder.WriteString("kdig")

	builder.WriteString(" @")
	builder.WriteString(req.Nameserver)
	builder.WriteString(" ")
	builder.WriteString(req.Name)
	builder.WriteString(" ")
	builder.WriteString(req.Type)
	switch strings.ToLower(req.Transport) {
	case "tcp":
		builder.WriteString(" +tcp")
	case "tls":
		builder.WriteString(" +tls")
	case "https":
		builder.WriteString(" +https")
	default:
		// UDP/default: no extra flags
	}
	if req.DNSSEC {
		builder.WriteString(" +dnssec +do")
	}
	if req.AsJSON {
		builder.WriteString(" +json")
	}
	if req.Short {
		builder.WriteString(" +short")
	}

	return builder.String()
}

// buildKdigArgs returns an args slice suitable for exec.Command, keeping
// flags as separate elements. The nameserver string is used verbatim
// (no http(s)/dns-query/port conversions) per project requirement.
func buildKdigArgs(req api.RequestPayload) []string {
	var args []string
	args = append(args, "@"+req.Nameserver)

	args = append(args, req.Name)
	args = append(args, req.Type)

	switch strings.ToLower(req.Transport) {
	case "tcp":
		args = append(args, "+tcp")
	case "tls":
		args = append(args, "+tls")
	case "https":
		args = append(args, "+https")
	default:
		// UDP/default: no transport flags
	}

	if req.DNSSEC {
		args = append(args, "+dnssec", "+do")
	}
	if req.AsJSON {
		args = append(args, "+json")
	}
	if req.Short {
		args = append(args, "+short")
	}

	return args
}

// BuildKdigArgsForTest exposes buildKdigArgs for tests in the external test package.
func BuildKdigArgsForTest(req api.RequestPayload) []string {
	return buildKdigArgs(req)
}

// BuildKdigCommandForTest exposes buildKdigCommand for tests in the external test package.
func BuildKdigCommandForTest(req api.RequestPayload) string {
	return buildKdigCommand(req)
}
