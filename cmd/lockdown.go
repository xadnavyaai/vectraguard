package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/vectra-guard/vectra-guard/internal/lockdown"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runLockdownEnable(ctx context.Context, reason string) error {
	logger := logging.FromContext(ctx)

	st, err := lockdown.GetState()
	if err != nil {
		return err
	}
	st.Enabled = true
	st.Reason = reason
	st.UpdatedBy = os.Getenv("USER")
	if err := lockdown.SetState(st); err != nil {
		return err
	}
	logger.Info("lockdown enabled", map[string]any{
		"reason":    st.Reason,
		"updatedBy": st.UpdatedBy,
	})
	fmt.Fprintln(os.Stderr, "🔒 Lockdown mode ENABLED: all guarded executions will be blocked until disabled.")
	return nil
}

func runLockdownDisable(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	st, err := lockdown.GetState()
	if err != nil {
		return err
	}
	st.Enabled = false
	st.Reason = ""
	if err := lockdown.SetState(st); err != nil {
		return err
	}
	logger.Info("lockdown disabled", nil)
	fmt.Fprintln(os.Stderr, "🔓 Lockdown mode DISABLED: guarded executions are allowed.")
	return nil
}

func runLockdownStatus(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	st, err := lockdown.GetState()
	if err != nil {
		return err
	}
	status := "DISABLED"
	if st.Enabled {
		status = "ENABLED"
	}
	logger.Info("lockdown status", map[string]any{
		"enabled":   st.Enabled,
		"reason":    st.Reason,
		"updatedBy": st.UpdatedBy,
	})
	fmt.Fprintf(os.Stdout, "Lockdown: %s\n", status)
	if st.Enabled && st.Reason != "" {
		fmt.Fprintf(os.Stdout, "Reason: %s\n", st.Reason)
	}
	return nil
}

