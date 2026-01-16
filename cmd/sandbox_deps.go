package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runSandboxDepsInstall(ctx context.Context, force bool, dryRun bool) error {
	fmt.Println("🛡️  Vectra Guard Sandbox Dependencies")
	fmt.Println("====================================")
	fmt.Println("")

	switch runtime.GOOS {
	case "darwin":
		return installSandboxDepsDarwin(ctx, force, dryRun)
	case "linux":
		return installSandboxDepsLinux(ctx, dryRun)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func installSandboxDepsDarwin(ctx context.Context, force bool, dryRun bool) error {
	logger := logging.FromContext(ctx)

	if !commandAvailable("brew") {
		return fmt.Errorf("Homebrew not found. Install Homebrew first: https://brew.sh/")
	}

	if fileExists("/usr/local/bin/hub-tool") {
		fmt.Println("⚠️  Detected /usr/local/bin/hub-tool (Docker Desktop conflict).")
		if !force {
			fmt.Println("   Skipping Docker Desktop install to avoid forcing changes.")
			fmt.Println("   You can install Docker Desktop manually if desired.")
			fmt.Println("")
			fmt.Println("   To force install (will remove the existing /usr/local/bin/hub-tool):")
			fmt.Println("     vectra-guard sandbox deps install --force")
			fmt.Println("   Or:")
			fmt.Println("     VG_FORCE=1 vectra-guard sandbox deps install")
			return nil
		}

		if shouldPrompt() {
			ok, err := promptConfirm("Proceed and remove /usr/local/bin/hub-tool? [y/N] ")
			if err != nil {
				return fmt.Errorf("prompt for confirmation: %w", err)
			}
			if !ok {
				return &exitError{message: "aborted by user", code: 1}
			}
		}

		if err := runCommand("sudo", []string{"rm", "-f", "/usr/local/bin/hub-tool"}, nil, dryRun); err != nil {
			return fmt.Errorf("remove /usr/local/bin/hub-tool: %w", err)
		}
	}

	envOverrides := []string{}
	if !isInteractive() {
		logger.Info("non-interactive session detected; disabling cask binary linking", nil)
		caskOpts := strings.TrimSpace(os.Getenv("HOMEBREW_CASK_OPTS"))
		if caskOpts != "" {
			caskOpts += " "
		}
		caskOpts += "--no-binaries"
		envOverrides = append(envOverrides, fmt.Sprintf("HOMEBREW_CASK_OPTS=%s", caskOpts))
	}

	fmt.Println("📦 Installing Docker Desktop...")
	if err := runCommand("brew", []string{"install", "--cask", "docker"}, envOverrides, dryRun); err != nil {
		if fileExists("/usr/local/bin/hub-tool") {
			fmt.Println("⚠️  Docker Desktop install failed due to /usr/local/bin/hub-tool conflict.")
			if force {
				fmt.Println("   Removing /usr/local/bin/hub-tool and retrying install.")
				if err := runCommand("sudo", []string{"rm", "-f", "/usr/local/bin/hub-tool"}, nil, dryRun); err != nil {
					return fmt.Errorf("remove /usr/local/bin/hub-tool: %w", err)
				}
				if err := runCommand("brew", []string{"install", "--cask", "docker"}, envOverrides, dryRun); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("remove /usr/local/bin/hub-tool or rerun with --force/VG_FORCE=1")
			}
		} else {
			return err
		}
	}

	fmt.Println("")
	fmt.Println("✅ Docker Desktop installed.")
	fmt.Println("   Open Docker.app once to finish setup.")
	return nil
}

func installSandboxDepsLinux(ctx context.Context, dryRun bool) error {
	logger := logging.FromContext(ctx)

	switch {
	case commandAvailable("apt-get"):
		fmt.Println("📦 Installing Docker + bubblewrap (Debian/Ubuntu)...")
		if err := runCommand("sudo", []string{"apt-get", "update", "-y"}, nil, dryRun); err != nil {
			return err
		}
		if err := runCommand("sudo", []string{"apt-get", "install", "-y", "docker.io", "docker-compose-plugin", "bubblewrap", "uidmap"}, nil, dryRun); err != nil {
			return err
		}
		if err := runCommand("sudo", []string{"systemctl", "enable", "--now", "docker"}, nil, dryRun); err != nil {
			return err
		}
		if user := os.Getenv("USER"); user != "" {
			_ = runCommand("sudo", []string{"usermod", "-aG", "docker", user}, nil, dryRun)
		}
	case commandAvailable("dnf"):
		fmt.Println("📦 Installing Podman + bubblewrap (Fedora/RHEL)...")
		if err := runCommand("sudo", []string{"dnf", "install", "-y", "podman", "podman-docker", "bubblewrap"}, nil, dryRun); err != nil {
			return err
		}
	case commandAvailable("yum"):
		fmt.Println("📦 Installing Podman + bubblewrap (CentOS/RHEL)...")
		if err := runCommand("sudo", []string{"yum", "install", "-y", "podman", "podman-docker", "bubblewrap"}, nil, dryRun); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported Linux package manager. Please install Docker/Podman and bubblewrap manually")
	}

	if commandAvailable("sysctl") {
		if err := runCommand("sudo", []string{"sysctl", "-w", "kernel.unprivileged_userns_clone=1"}, nil, dryRun); err != nil {
			logger.Warn("unable to enable unprivileged user namespaces (continuing)", map[string]any{"error": err.Error()})
		}
	}

	fmt.Println("")
	fmt.Println("✅ Sandbox dependencies installed.")
	fmt.Println("   Log out and back in to apply docker group changes.")
	return nil
}

func runCommand(name string, args []string, envOverrides []string, dryRun bool) error {
	if dryRun {
		fmt.Printf("→ %s %s\n", name, strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if len(envOverrides) > 0 {
		cmd.Env = append(os.Environ(), envOverrides...)
	}
	return cmd.Run()
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isInteractive() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

func isTerminal(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func shouldPrompt() bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = tty.Close()
	return true
}

func promptConfirm(prompt string) (bool, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false, nil
	}
	defer tty.Close()

	if _, err := fmt.Fprint(tty, prompt); err != nil {
		return false, err
	}
	reader := bufio.NewReader(tty)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return false, nil
	}
	return strings.HasPrefix(strings.ToLower(input), "y"), nil
}

func envBool(key string) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
