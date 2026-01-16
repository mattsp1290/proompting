package resume

import (
	"strings"
	"testing"
	"time"

	"github.com/vibes-project/vibes/internal/beads"
	"github.com/vibes-project/vibes/internal/git"
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

func TestGetPendingItems(t *testing.T) {
	t.Run("includes stash warning", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 1 && args[0] == "stash" {
					return "stash@{0}: WIP on feature: abc123 Some work", nil
				}
				return "", nil
			},
		}

		task := beads.TaskInfo{ID: "bd-123"}
		items := getPendingItems("/test/dir", task, mock, false)

		hasStashWarning := false
		for _, item := range items {
			if strings.Contains(item, "stashed") {
				hasStashWarning = true
				break
			}
		}
		if !hasStashWarning {
			t.Error("expected stash warning in pending items")
		}
	})

	t.Run("includes inbox hint when task has ID", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		task := beads.TaskInfo{ID: "bd-123"}
		items := getPendingItems("/test/dir", task, mock, false)

		hasInboxHint := false
		for _, item := range items {
			if strings.Contains(item, "bd-123-review") {
				hasInboxHint = true
				break
			}
		}
		if !hasInboxHint {
			t.Error("expected inbox hint in pending items")
		}
	})

	t.Run("detects behind remote", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "## feature/test...origin/feature/test [behind 3]", nil
				}
				return "", nil
			},
		}

		task := beads.TaskInfo{}
		items := getPendingItems("/test/dir", task, mock, false)

		hasBehindWarning := false
		for _, item := range items {
			if strings.Contains(item, "behind") {
				hasBehindWarning = true
				break
			}
		}
		if !hasBehindWarning {
			t.Error("expected behind warning in pending items")
		}
	})

	t.Run("detects ahead of remote", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "## feature/test...origin/feature/test [ahead 2]", nil
				}
				return "", nil
			},
		}

		task := beads.TaskInfo{}
		items := getPendingItems("/test/dir", task, mock, false)

		hasAheadNotice := false
		for _, item := range items {
			if strings.Contains(item, "ahead") {
				hasAheadNotice = true
				break
			}
		}
		if !hasAheadNotice {
			t.Error("expected ahead notice in pending items")
		}
	})
}

func TestGetProtocol(t *testing.T) {
	task := beads.TaskInfo{ID: "bd-123", Title: "Test task", Branch: "feature/test", ProjectName: "my-project"}

	t.Run("non-verbose protocol", func(t *testing.T) {
		result := getProtocol(task, false)

		if !strings.Contains(result, "vibes done") {
			t.Error("expected vibes done reference")
		}
		if !strings.Contains(result, "Check inbox") {
			t.Error("expected inbox check instruction")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getProtocol(task, true)

		if !strings.Contains(result, "**Check for updates**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "file_reservation_paths") {
			t.Error("expected MCP function reference")
		}
		if !strings.Contains(result, "```bash") {
			t.Error("expected code blocks in verbose mode")
		}
		if !strings.Contains(result, "bd show bd-123") {
			t.Error("expected task ID in show command")
		}
		if !strings.Contains(result, "project_key=\"my-project\"") {
			t.Error("expected project name in project_key")
		}
		if !strings.Contains(result, "<your-file-patterns>") {
			t.Error("expected proper placeholder in patterns")
		}
	})

	t.Run("uses placeholder when no task ID", func(t *testing.T) {
		emptyTask := beads.TaskInfo{}
		result := getProtocol(emptyTask, true)

		if !strings.Contains(result, "<task-id>") {
			t.Error("expected placeholder when no task ID")
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
				if command == "git" && len(args) >= 1 && args[0] == "stash" {
					return "", nil
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

	t.Run("with no-fetch option", func(t *testing.T) {
		tmpDir := t.TempDir()
		fetchCalled := false

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 1 && args[0] == "fetch" {
					fetchCalled = true
				}
				return "", nil
			},
		}

		opts := Options{
			Dir:     tmpDir,
			NoFetch: true,
			Runner:  mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if fetchCalled {
			t.Error("fetch should not be called when NoFetch is true")
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

// Test that shared packages are used correctly
func TestSharedPackageIntegration(t *testing.T) {
	t.Run("git.CountLines works", func(t *testing.T) {
		result := git.CountLines("line1\nline2\nline3")
		if result != 3 {
			t.Errorf("expected 3, got %d", result)
		}
	})

	t.Run("beads.ExtractIDFromBranch works", func(t *testing.T) {
		result := beads.ExtractIDFromBranch("feature/bd-123-test")
		if result != "bd-123" {
			t.Errorf("expected bd-123, got %s", result)
		}
	})
}
