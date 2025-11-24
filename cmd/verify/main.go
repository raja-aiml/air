package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	path, err := exec.LookPath("air")
	if err != nil {
		fmt.Fprintln(os.Stderr, "'air' binary not found in PATH. Build or install the main CLI and retry.")
		os.Exit(2)
	}

	args := append([]string{"verify"}, os.Args[1:]...)
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		fmt.Fprintln(os.Stderr, "failed to run 'air verify':", err)
		os.Exit(1)
	}
}
