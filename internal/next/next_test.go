package next

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// MockRunner is a mock implementation of CommandRunner for testing
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

	t.Run("repo with modified files", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "main", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					// Note: DefaultRunner.Run returns TrimSpace'd output
					// " M file.go" after TrimSpace becomes "M file.go" for first line only
					// Use MM to indicate both staged and modified in worktree
					return "MM file1.go\nMM file2.go\nMM file3.go", nil
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Fix bug (30 minutes ago)", nil
				}
				return "", nil
			},
		}

		result := getGitContext("/test/dir", mock)

		if !strings.Contains(result, "3 modified") {
			t.Errorf("expected 3 modified files, got: %s", result)
		}
	})

	t.Run("repo with untracked files", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "main", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "?? untracked1.go\n?? untracked2.go", nil
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Init (1 day ago)", nil
				}
				return "", nil
			},
		}

		result := getGitContext("/test/dir", mock)

		if !strings.Contains(result, "2 untracked") {
			t.Errorf("expected 2 untracked files, got: %s", result)
		}
	})

	t.Run("repo with mixed status", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "develop", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "A  staged.go\n M modified.go\n?? untracked.go", nil
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Merge PR (5 minutes ago)", nil
				}
				return "", nil
			},
		}

		result := getGitContext("/test/dir", mock)

		if !strings.Contains(result, "1 staged") {
			t.Errorf("expected 1 staged, got: %s", result)
		}
		if !strings.Contains(result, "1 modified") {
			t.Errorf("expected 1 modified, got: %s", result)
		}
		if !strings.Contains(result, "1 untracked") {
			t.Errorf("expected 1 untracked, got: %s", result)
		}
	})

	t.Run("git commands fail", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", errors.New("git not found")
			},
		}

		result := getGitContext("/test/dir", mock)

		// Should still return clean status when git status fails (empty string)
		if !strings.Contains(result, "Clean working tree") {
			t.Errorf("expected clean working tree on git failure, got: %s", result)
		}
	})

	t.Run("short status line", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "main", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "X", nil // Too short, should be skipped
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Commit (now)", nil
				}
				return "", nil
			},
		}

		result := getGitContext("/test/dir", mock)

		// Should not crash and should show no status parts
		if strings.Contains(result, "staged") || strings.Contains(result, "modified") || strings.Contains(result, "untracked") {
			t.Errorf("expected no status counts for short line, got: %s", result)
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
				return "", errors.New("not found")
			},
		}

		result := getTaskRecommendation(tmpDir, mock)

		if !strings.Contains(result, "Task 1: Fix bug") {
			t.Errorf("expected bv output, got: %s", result)
		}
	})

	t.Run("beads directory with bv failure, bd success", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "bv" {
					return "", errors.New("bv not found")
				}
				if command == "bd" {
					return "bd-001: Ready task", nil
				}
				return "", errors.New("not found")
			},
		}

		result := getTaskRecommendation(tmpDir, mock)

		if !strings.Contains(result, "bd-001: Ready task") {
			t.Errorf("expected bd output, got: %s", result)
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
				return "", errors.New("command not found")
			},
		}

		result := getTaskRecommendation(tmpDir, mock)

		if !strings.Contains(result, "no ready tasks found") {
			t.Errorf("expected fallback message, got: %s", result)
		}
	})

	t.Run("beads directory with empty output", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := filepath.Join(tmpDir, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", nil // Success but empty output
			},
		}

		result := getTaskRecommendation(tmpDir, mock)

		if !strings.Contains(result, "no ready tasks found") {
			t.Errorf("expected fallback message for empty output, got: %s", result)
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
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "main", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "", nil
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Test commit (now)", nil
				}
				return "", nil
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Runner: mock,
		}

		// Capture stdout would require more setup, just verify no error
		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with default directory", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if dir != cwd {
					t.Errorf("expected dir %s, got %s", cwd, dir)
				}
				return "", nil
			},
		}

		opts := Options{
			Runner: mock,
		}

		err = Run(opts)
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
			Runner: nil, // Should use DefaultRunner
		}

		// This will try to run real git commands, which may fail in tmpDir
		// but should not panic
		_ = Run(opts)
	})
}

func TestDefaultRunner(t *testing.T) {
	runner := &DefaultRunner{}

	t.Run("Run with valid command", func(t *testing.T) {
		result, err := runner.Run(".", "echo", "hello")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "hello" {
			t.Errorf("expected 'hello', got: %s", result)
		}
	})

	t.Run("Run with invalid command", func(t *testing.T) {
		_, err := runner.Run(".", "nonexistent-command-12345")
		if err == nil {
			t.Error("expected error for nonexistent command")
		}
	})

	t.Run("RunWithTimeout with valid command", func(t *testing.T) {
		result, err := runner.RunWithTimeout(".", 5*time.Second, "echo", "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "test" {
			t.Errorf("expected 'test', got: %s", result)
		}
	})

	t.Run("RunWithTimeout with command not in PATH", func(t *testing.T) {
		_, err := runner.RunWithTimeout(".", 5*time.Second, "nonexistent-command-12345")
		if err == nil {
			t.Error("expected error for command not in PATH")
		}
	})

	t.Run("RunWithTimeout respects directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		result, err := runner.RunWithTimeout(tmpDir, 5*time.Second, "pwd")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != tmpDir {
			t.Errorf("expected %s, got: %s", tmpDir, result)
		}
	})
}

func TestGitStatusParsing(t *testing.T) {
	testCases := []struct {
		name     string
		status   string
		expected struct {
			staged    int
			modified  int
			untracked int
		}
	}{
		{
			name:   "staged only",
			status: "A  file1.go\nM  file2.go",
			expected: struct {
				staged    int
				modified  int
				untracked int
			}{2, 0, 0},
		},
		{
			name:   "modified only",
			status: "AM file1.go\nAM file2.go", // A=staged, M=modified in worktree
			expected: struct {
				staged    int
				modified  int
				untracked int
			}{2, 2, 0}, // Both staged and modified
		},
		{
			name:   "untracked only",
			status: "?? file1.go\n?? file2.go\n?? file3.go",
			expected: struct {
				staged    int
				modified  int
				untracked int
			}{0, 0, 3},
		},
		{
			name:   "mixed",
			status: "A  staged.go\n M modified.go\n?? untracked.go\nMM both.go",
			expected: struct {
				staged    int
				modified  int
				untracked int
			}{2, 2, 1}, // MM counts as 1 staged and 1 modified
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockRunner{
				RunFunc: func(dir string, command string, args ...string) (string, error) {
					if len(args) >= 1 && args[0] == "status" {
						return tc.status, nil
					}
					return "", nil
				},
			}

			result := getGitContext("/test", mock)

			if tc.expected.staged > 0 && !strings.Contains(result, fmt.Sprintf("%d staged", tc.expected.staged)) {
				t.Errorf("expected %d staged in result: %s", tc.expected.staged, result)
			}
			if tc.expected.modified > 0 && !strings.Contains(result, fmt.Sprintf("%d modified", tc.expected.modified)) {
				t.Errorf("expected %d modified in result: %s", tc.expected.modified, result)
			}
			if tc.expected.untracked > 0 && !strings.Contains(result, fmt.Sprintf("%d untracked", tc.expected.untracked)) {
				t.Errorf("expected %d untracked in result: %s", tc.expected.untracked, result)
			}
		})
	}
}
