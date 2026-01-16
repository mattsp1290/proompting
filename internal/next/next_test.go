package next

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vibes-project/vibes/internal/runner"
)

// MockRunner is a mock implementation of runner.CommandRunner for testing
type MockRunner struct {
	RunFunc            func(dir string, command string, args ...string) (string, error)
	RunWithTimeoutFunc func(dir string, timeout time.Duration, command string, args ...string) (string, error)
}

func (m *MockRunner) Run(dir string, command string, args ...string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(dir, command, args...)
	}
	return "", nil
}

func (m *MockRunner) RunWithTimeout(dir string, timeout time.Duration, command string, args ...string) (string, error) {
	if m.RunWithTimeoutFunc != nil {
		return m.RunWithTimeoutFunc(dir, timeout, command, args...)
	}
	return "", nil
}

func TestGetProtocol(t *testing.T) {
	t.Run("non-verbose protocol", func(t *testing.T) {
		result := getProtocol(false)

		if !strings.Contains(result, "Claim:") {
			t.Error("expected non-verbose protocol to contain 'Claim:'")
		}
		if !strings.Contains(result, "`bd update <id> --status in_progress`") {
			t.Error("expected non-verbose protocol to contain inline command")
		}
		if strings.Contains(result, "**Claim the work**") {
			t.Error("non-verbose protocol should not contain bold headers")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getProtocol(true)

		if !strings.Contains(result, "**Claim the work**") {
			t.Error("expected verbose protocol to contain bold headers")
		}
		if !strings.Contains(result, "```bash") {
			t.Error("expected verbose protocol to contain code blocks")
		}
		if !strings.Contains(result, "file_reservation_paths") {
			t.Error("expected verbose protocol to contain MCP function call")
		}
		if !strings.Contains(result, "<your-file-patterns>") {
			t.Error("expected proper placeholder in patterns")
		}
		if !strings.Contains(result, "Begin working on the highest priority task now.") {
			t.Error("expected verbose protocol to end with call to action")
		}
	})
}

func TestGetGitContext(t *testing.T) {
	t.Run("clean repo with branch and commit", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "main", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "", nil // clean
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Initial commit (2 hours ago)", nil
				}
				return "", nil
			},
		}

		result := getGitContext("/test/dir", mock)

		if !strings.Contains(result, "**Branch**: main") {
			t.Errorf("expected branch main, got: %s", result)
		}
		if !strings.Contains(result, "Clean working tree") {
			t.Errorf("expected clean status, got: %s", result)
		}
		if !strings.Contains(result, "Initial commit (2 hours ago)") {
			t.Errorf("expected recent commit, got: %s", result)
		}
	})

	t.Run("repo with staged files", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "feature/test", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "A  newfile.go\nM  modified.go", nil
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Add feature (1 hour ago)", nil
				}
				return "", nil
			},
		}

		result := getGitContext("/test/dir", mock)

		if !strings.Contains(result, "2 staged") {
			t.Errorf("expected 2 staged files, got: %s", result)
		}
	})
}

func TestGetTaskRecommendation(t *testing.T) {
	t.Run("no beads directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{}

		result := getTaskRecommendation(tmpDir, mock)

		if result != "" {
			t.Errorf("expected empty result when no .beads dir, got: %s", result)
		}
	})

	t.Run("beads directory with bv success", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "bv" {
					return "Task 1: Fix bug\nTask 2: Add feature", nil
				}
				return "", nil
			},
		}

		result := getTaskRecommendation(tmpDir, mock)

		if !strings.Contains(result, "Task 1: Fix bug") {
			t.Errorf("expected bv output, got: %s", result)
		}
	})

	t.Run("beads directory with both commands failing", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", nil // Empty output
			},
		}

		result := getTaskRecommendation(tmpDir, mock)

		if !strings.Contains(result, "no ready tasks found") {
			t.Errorf("expected fallback message, got: %s", result)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("with specified directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if dir != tmpDir {
					t.Errorf("expected dir %s, got %s", tmpDir, dir)
				}
				return "", nil
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Runner: mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("verbose mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:     tmpDir,
			Verbose: true,
			Runner:  mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with nil runner uses default", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := Options{
			Dir:    tmpDir,
			Runner: nil,
		}

		// Should not panic
		_ = Run(opts)
	})
}

func TestDefaultRunner(t *testing.T) {
	r := &runner.Default{}

	t.Run("Run with valid command", func(t *testing.T) {
		result, err := r.Run(".", "echo", "hello")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "hello" {
			t.Errorf("expected 'hello', got: %s", result)
		}
	})

	t.Run("Run with invalid command", func(t *testing.T) {
		_, err := r.Run(".", "nonexistent-command-12345")
		if err == nil {
			t.Error("expected error for nonexistent command")
		}
	})
}
