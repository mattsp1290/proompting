package beads

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestIsInitialized(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		if IsInitialized(tmpDir) {
			t.Error("expected false for uninitialized directory")
		}
	})

	t.Run("initialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		if !IsInitialized(tmpDir) {
			t.Error("expected true for initialized directory")
		}
	})
}

func TestExtractIDFromBranch(t *testing.T) {
	testCases := []struct {
		branch   string
		expected string
	}{
		{"feature/bd-123-add-feature", "bd-123"},
		{"bd-456", "bd-456"},
		{"feature/BEAD-789-fix", "BEAD-789"},
		{"main", ""},
		{"feature/some-feature", ""},
		{"bd-123", "bd-123"},
		{"hotfix/bead-42-urgent", "bead-42"},
	}

	for _, tc := range testCases {
		t.Run(tc.branch, func(t *testing.T) {
			result := ExtractIDFromBranch(tc.branch)
			if result != tc.expected {
				t.Errorf("ExtractIDFromBranch(%q) = %q, want %q", tc.branch, result, tc.expected)
			}
		})
	}
}

func TestParseListLine(t *testing.T) {
	testCases := []struct {
		line          string
		expectedID    string
		expectedTitle string
	}{
		{"bd-123  Some task title  [in_progress]", "bd-123", "Some task title"},
		{"bd-456  Another task", "bd-456", "Another task"},
		{"bd-789", "bd-789", ""},
		{"", "", ""},
		{"not a bead line", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			id, title := ParseListLine(tc.line)
			if id != tc.expectedID {
				t.Errorf("ParseListLine(%q) id = %q, want %q", tc.line, id, tc.expectedID)
			}
			if title != tc.expectedTitle {
				t.Errorf("ParseListLine(%q) title = %q, want %q", tc.line, title, tc.expectedTitle)
			}
		})
	}
}

func TestExtractTitleFromShow(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected string
	}{
		{"with title", "Title: Some task\nStatus: in_progress\nPriority: 1", "Some task"},
		{"no title", "Status: in_progress\nPriority: 1", ""},
		{"empty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractTitleFromShow(tc.output)
			if result != tc.expected {
				t.Errorf("ExtractTitleFromShow() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestExtractStatusFromShow(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected string
	}{
		{"with status", "Title: Some task\nStatus: in_progress\nPriority: 1", "in_progress"},
		{"no status", "Title: Some task\nPriority: 1", ""},
		{"empty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractStatusFromShow(tc.output)
			if result != tc.expected {
				t.Errorf("ExtractStatusFromShow() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestDetectCurrentTask(t *testing.T) {
	t.Run("no beads directory uses branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{}

		task := DetectCurrentTask(tmpDir, "feature/bd-123-test", mock)

		if task.ID != "bd-123" {
			t.Errorf("expected ID 'bd-123', got %q", task.ID)
		}
	})

	t.Run("with beads finds in-progress task", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "bd" && len(args) >= 2 && args[0] == "list" {
					return "bd-456  Working on feature  [in_progress]", nil
				}
				return "", nil
			},
		}

		task := DetectCurrentTask(tmpDir, "feature/test", mock)

		if task.ID != "bd-456" {
			t.Errorf("expected ID 'bd-456', got %q", task.ID)
		}
		if task.Title != "Working on feature" {
			t.Errorf("expected title 'Working on feature', got %q", task.Title)
		}
		if task.Status != "in_progress" {
			t.Errorf("expected status 'in_progress', got %q", task.Status)
		}
	})

	t.Run("falls back to branch when bd list fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", errors.New("bd not found")
			},
		}

		task := DetectCurrentTask(tmpDir, "feature/bd-789-fallback", mock)

		if task.ID != "bd-789" {
			t.Errorf("expected ID 'bd-789', got %q", task.ID)
		}
	})

	t.Run("gets title and status from bd show", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "bd" && args[0] == "list" {
					return "", nil // No in-progress tasks
				}
				if command == "bd" && args[0] == "show" {
					return "Title: Task from branch\nStatus: open\nPriority: 1", nil
				}
				return "", nil
			},
		}

		task := DetectCurrentTask(tmpDir, "feature/bd-999-from-branch", mock)

		if task.ID != "bd-999" {
			t.Errorf("expected ID 'bd-999', got %q", task.ID)
		}
		if task.Title != "Task from branch" {
			t.Errorf("expected title 'Task from branch', got %q", task.Title)
		}
		if task.Status != "open" {
			t.Errorf("expected status 'open', got %q", task.Status)
		}
	})
}
