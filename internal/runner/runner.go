package runner

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner executes shell commands
type CommandRunner interface {
	Run(dir string, command string, args ...string) (string, error)
	RunWithTimeout(dir string, timeout time.Duration, command string, args ...string) (string, error)
}

// Default is the default command runner that executes real commands
type Default struct{}

// Run executes a command and returns stdout
func (r *Default) Run(dir string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunWithTimeout executes a command with a timeout
func (r *Default) RunWithTimeout(dir string, timeout time.Duration, command string, args ...string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}
