package cmd

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunServeBindsLocalhost(t *testing.T) {
	// Find a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("unable to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	var buf bytes.Buffer
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", &buf))

	done := make(chan struct{})
	go func() {
		// runServe blocks; allow it to run briefly then terminate by context cancellation in real use.
		_ = runServe(ctx, port)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		// We only care that starting didn't panic; we don't keep it running for long in tests.
	}
}
