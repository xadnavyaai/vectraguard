package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/serve"
)

func runServe(ctx context.Context, port int) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	workspace, err := serve.DefaultWorkspace()
	if err != nil {
		return err
	}

	srv, err := serve.New(workspace, logger, cfg.Sandbox.EnableMetrics)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Serving Vectra Guard dashboard at http://127.0.0.1:%d\n", port)
	return srv.ListenAndServe(port)
}

// parsePortEnv is a small helper for tests / future flags.
func parsePortEnv(envVal string, defaultPort int) int {
	if envVal == "" {
		return defaultPort
	}
	if p, err := strconv.Atoi(envVal); err == nil && p > 0 && p < 65536 {
		return p
	}
	return defaultPort
}

