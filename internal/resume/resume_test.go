package resume

import (
	"errors"
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

func TestExtractBeadIDFromBranch(t *testing.T) {
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
			result := extractBeadIDFromBranch(tc.branch)
			if result != tc.expected {
				t.Errorf("extractBeadIDFromBranch(%q) = %q, want %q", tc.branch, result, tc.expected)
			}
		})
	}
}

func TestParseBeadLine(t *testing.T) {
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
			id, title := parseBeadLine(tc.line)
			if id != tc.expectedID {
				t.Errorf("parseBeadLine(%q) id = %q, want %q", tc.line, id, tc.expectedID)
			}
			if title != tc.expectedTitle {
				t.Errorf("parseBeadLine(%q) title = %q, want %q", tc.line, title, tc.expectedTitle)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("returns branch name", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" {
					return "feature/bd-123-test", nil
				}
				return "", nil
			},
		}

		result := getCurrentBranch("/test/dir", mock)
		if result != "feature/bd-123-test" {
			t.Errorf("expected 'feature/bd-123-test', got %q", result)
		}
	})

	t.Run("returns empty on error", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", errors.New("not a git repo")
			},
		}

		result := getCurrentBranch("/test/dir", mock)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestGetUncommittedChanges(t *testing.T) {
	testCases := []struct {
		name     string
		status   string
		expected string
	}{
		{"clean", "", ""},
		{"staged only", "A  file.go", "1 staged"},
		{"modified only", "MM file.go", "1 staged, 1 modified"},
		{"untracked only", "?? file.go", "1 untracked"},
		{"mixed", "A  a.go\n M b.go\n?? c.go", "1 staged, 1 modified, 1 untracked"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockRunner{
				RunFunc: func(dir string, command string, args ...string) (string, error) {
					return tc.status, nil
				},
			}

			result := getUncommittedChanges("/test/dir", mock)
			if result != tc.expected {
				t.Errorf("getUncommittedChanges() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestGetBranchCommits(t *testing.T) {
	t.Run("feature branch with commits", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 3 && args[2] == "main..HEAD" {
					return "abc123 Add feature\ndef456 Fix bug", nil
				}
				return "", nil
			},
		}

		result := getBranchCommits("/test/dir", "feature/test", mock)
		if !strings.Contains(result, "abc123 Add feature") {
			t.Errorf("expected commits, got %q", result)
		}
	})

	t.Run("main branch shows recent commits", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "log" && args[1] == "-5" {
					return "abc123 Recent commit\ndef456 Older commit", nil
				}
				return "", nil
			},
		}

		result := getBranchCommits("/test/dir", "main", mock)
		if !strings.Contains(result, "abc123 Recent commit") {
			t.Errorf("expected recent commits for main, got %q", result)
		}
	})

	t.Run("falls back to master if main fails", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 3 {
					if args[2] == "main..HEAD" {
						return "", errors.New("main not found")
					}
					if args[2] == "master..HEAD" {
						return "abc123 Commit from master", nil
					}
				}
				return "", nil
			},
		}

		result := getBranchCommits("/test/dir", "feature/test", mock)
		if !strings.Contains(result, "abc123 Commit from master") {
			t.Errorf("expected master fallback, got %q", result)
		}
	})
}

func TestCountLines(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"one line", 1},
		{"line1\nline2", 2},
		{"line1\nline2\nline3", 3},
		{"line1\nline2\n", 2},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := countLines(tc.input)
			if result != tc.expected {
				t.Errorf("countLines(%q) = %d, want %d", tc.input, result, tc.expected)
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
			result := extractStatusFromShow(tc.output)
			if result != tc.expected {
				t.Errorf("extractStatusFromShow() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestDetectCurrentTask(t *testing.T) {
	t.Run("no beads directory uses branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{}

		task := detectCurrentTask(tmpDir, "feature/bd-123-test", mock)

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

		task := detectCurrentTask(tmpDir, "feature/test", mock)

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

		task := detectCurrentTask(tmpDir, "feature/bd-789-fallback", mock)

		if task.ID != "bd-789" {
			t.Errorf("expected ID 'bd-789', got %q", task.ID)
		}
	})
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

		task := TaskInfo{ID: "bd-123"}
		items := getPendingItems("/test/dir", task, mock)

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

		task := TaskInfo{ID: "bd-123"}
		items := getPendingItems("/test/dir", task, mock)

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
}

func TestCheckRemoteStatus(t *testing.T) {
	t.Run("detects behind remote", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "## feature/test...origin/feature/test [behind 3]", nil
				}
				return "", nil
			},
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		result := checkRemoteStatus("/test/dir", mock)
		if !strings.Contains(result, "behind") {
			t.Errorf("expected behind warning, got %q", result)
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
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		result := checkRemoteStatus("/test/dir", mock)
		if !strings.Contains(result, "ahead") {
			t.Errorf("expected ahead notice, got %q", result)
		}
	})
}

func TestGetProtocol(t *testing.T) {
	task := TaskInfo{ID: "bd-123", Title: "Test task", Branch: "feature/test", ProjectName: "my-project"}

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
	})

	t.Run("uses placeholder when no task ID", func(t *testing.T) {
		emptyTask := TaskInfo{}
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
