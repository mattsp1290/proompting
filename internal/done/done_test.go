package done

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
	task := beads.TaskInfo{ID: "bd-123", Title: "Test task", Branch: "feature/test", ProjectName: "my-project"}

	t.Run("non-verbose protocol", func(t *testing.T) {
		result := getProtocol(task, false)

		if !strings.Contains(result, "bd update bd-123 --status closed") {
			t.Error("expected task ID in completion command")
		}
		if !strings.Contains(result, "vibes next") {
			t.Error("expected vibes next reference")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getProtocol(task, true)

		if !strings.Contains(result, "**Verify work is complete**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "release_file_paths") {
			t.Error("expected MCP function reference")
		}
		if !strings.Contains(result, "```bash") {
			t.Error("expected code blocks in verbose mode")
		}
		if !strings.Contains(result, "project_key=\"my-project\"") {
			t.Error("expected project name in project_key")
		}
	})

	t.Run("uses placeholder when no task ID", func(t *testing.T) {
		emptyTask := beads.TaskInfo{}
		result := getProtocol(emptyTask, false)

		if !strings.Contains(result, "<task-id>") {
			t.Error("expected placeholder when no task ID")
		}
	})

	t.Run("uses default project-name when no project name", func(t *testing.T) {
		taskNoProject := beads.TaskInfo{ID: "bd-456"}
		result := getProtocol(taskNoProject, true)

		if !strings.Contains(result, "project_key=\"project-name\"") {
			t.Error("expected default project-name when no project name set")
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
