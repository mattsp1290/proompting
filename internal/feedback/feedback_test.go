package feedback

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

		if !strings.Contains(result, "bd-123-review") {
			t.Error("expected task ID review thread reference")
		}
		if !strings.Contains(result, "vibes pr") {
			t.Error("expected vibes pr reference")
		}
		if !strings.Contains(result, "blocking > suggestions > questions > nitpicks") {
			t.Error("expected triage order")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getProtocol(task, true)

		if !strings.Contains(result, "**Retrieve review feedback**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "file_reservation_paths") {
			t.Error("expected MCP function reference")
		}
		if !strings.Contains(result, "send_message") {
			t.Error("expected send_message function reference")
		}
		if !strings.Contains(result, "project_key=\"my-project\"") {
			t.Error("expected project name in project_key")
		}
		if !strings.Contains(result, "Bead: bd-123") {
			t.Error("expected bead reference in commit message")
		}
	})

	t.Run("uses placeholder when no task ID", func(t *testing.T) {
		emptyTask := beads.TaskInfo{}
		result := getProtocol(emptyTask, false)

		if !strings.Contains(result, "<task-id>-review") {
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

func TestGetInboxHint(t *testing.T) {
	task := beads.TaskInfo{ID: "bd-123", ProjectName: "my-project"}

	t.Run("non-verbose hint", func(t *testing.T) {
		result := getInboxHint(task, false)

		if !strings.Contains(result, "resource://inbox/YourAgentIdentity") {
			t.Error("expected inbox resource reference")
		}
		if !strings.Contains(result, "bd-123-review") {
			t.Error("expected thread ID")
		}
	})

	t.Run("verbose hint", func(t *testing.T) {
		result := getInboxHint(task, true)

		if !strings.Contains(result, "get_thread_messages") {
			t.Error("expected get_thread_messages function")
		}
		if !strings.Contains(result, "project_key=\"my-project\"") {
			t.Error("expected project key")
		}
		if !strings.Contains(result, "**Blocking**") {
			t.Error("expected feedback categories")
		}
	})

	t.Run("uses placeholder when no task ID", func(t *testing.T) {
		emptyTask := beads.TaskInfo{}
		result := getInboxHint(emptyTask, false)

		if !strings.Contains(result, "<task-id>-review") {
			t.Error("expected placeholder thread ID")
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
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "feature/bd-123-test", nil
				}
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--verify" {
					return "main", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "log" {
					return "abc123 Test commit", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "diff" {
					return "3 files changed, 100 insertions(+), 10 deletions(-)", nil
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

func TestGetBaseBranch(t *testing.T) {
	t.Run("returns main when main exists", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[1] == "main" {
					return "main", nil
				}
				return "", nil
			},
		}

		result := getBaseBranch("/tmp", mock)
		if result != "main" {
			t.Errorf("expected main, got %s", result)
		}
	})
}

func TestGetDiffStats(t *testing.T) {
	t.Run("returns summary line", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return " file1.go | 10 ++++\n file2.go | 5 --\n 2 files changed, 10 insertions(+), 5 deletions(-)", nil
			},
		}

		result := getDiffStats("/tmp", "main", mock)
		if !strings.Contains(result, "2 files changed") {
			t.Errorf("expected summary line, got %s", result)
		}
	})

	t.Run("returns empty on error", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		result := getDiffStats("/tmp", "main", mock)
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})
}
