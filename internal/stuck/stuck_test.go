package stuck

import (
	"strings"
	"testing"
	"time"

	"github.com/vibes-project/vibes/internal/beads"
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

		if !strings.Contains(result, "Analyze the recent changes") {
			t.Error("expected analysis step")
		}
		if !strings.Contains(result, "Diagnose the root cause") {
			t.Error("expected diagnosis step")
		}
		if !strings.Contains(result, "Propose a specific fix") {
			t.Error("expected fix proposal step")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getProtocol(true)

		if !strings.Contains(result, "**Analyze the situation**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "Typos or syntax errors") {
			t.Error("expected detailed diagnosis hints")
		}
		if !strings.Contains(result, "If still stuck") {
			t.Error("expected fallback advice")
		}
	})
}

func TestTruncateOutput(t *testing.T) {
	t.Run("short output unchanged", func(t *testing.T) {
		input := "line1\nline2\nline3"
		result := truncateOutput(input, 10)
		if result != input {
			t.Errorf("expected unchanged output, got %s", result)
		}
	})

	t.Run("long output truncated", func(t *testing.T) {
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = "line"
		}
		input := strings.Join(lines, "\n")
		result := truncateOutput(input, 5)

		if !strings.Contains(result, "... (15 more lines)") {
			t.Error("expected truncation message")
		}
		outputLines := strings.Split(result, "\n")
		if len(outputLines) != 6 { // 5 lines + truncation message
			t.Errorf("expected 6 lines, got %d", len(outputLines))
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
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" {
					return "feature/bd-123-test", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "log" {
					return "abc123 Test commit", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "diff" {
					return "+added line\n-removed line", nil
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

	t.Run("with description", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:         tmpDir,
			Description: "tests are failing mysteriously",
			Runner:      mock,
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

func TestFileExists(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		if fileExists(tmpDir) != true {
			t.Error("expected directory to exist")
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		if fileExists("/nonexistent/path/to/file") != false {
			t.Error("expected file to not exist")
		}
	})
}

// Verify TaskInfo is used correctly (compile-time check)
var _ = beads.TaskInfo{}
