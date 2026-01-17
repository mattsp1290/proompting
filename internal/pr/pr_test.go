package pr

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
		result := getProtocol(task, "main", false)

		if !strings.Contains(result, "gh pr create --base main") {
			t.Error("expected gh pr create command with base branch")
		}
		if !strings.Contains(result, "Review changes") {
			t.Error("expected review changes step")
		}
		if !strings.Contains(result, "bd-123") {
			t.Error("expected task reference")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getProtocol(task, "main", true)

		if !strings.Contains(result, "**Review changes**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "gh pr create --base main") {
			t.Error("expected gh pr create command")
		}
		if !strings.Contains(result, "```bash") {
			t.Error("expected code blocks in verbose mode")
		}
		if !strings.Contains(result, "Security vulnerabilities") {
			t.Error("expected security check in verbose mode")
		}
		if !strings.Contains(result, "gh pr view --web") {
			t.Error("expected verification step in verbose mode")
		}
	})

	t.Run("includes task context when available", func(t *testing.T) {
		result := getProtocol(task, "main", false)

		if !strings.Contains(result, "bd-123") {
			t.Error("expected task ID in protocol")
		}
		if !strings.Contains(result, "Test task") {
			t.Error("expected task title in protocol")
		}
	})

	t.Run("works without task context", func(t *testing.T) {
		emptyTask := beads.TaskInfo{}
		result := getProtocol(emptyTask, "main", false)

		if !strings.Contains(result, "gh pr create") {
			t.Error("expected gh pr create even without task")
		}
	})

	t.Run("uses correct base branch", func(t *testing.T) {
		result := getProtocol(task, "master", false)

		if !strings.Contains(result, "gh pr create --base master") {
			t.Error("expected master as base branch")
		}
	})
}

func TestGetBaseBranch(t *testing.T) {
	t.Run("returns main when main exists", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 3 && args[2] == "main" {
					return "abc123", nil
				}
				return "", nil
			},
		}

		result := getBaseBranch("/test", mock)
		if result != "main" {
			t.Errorf("expected main, got %s", result)
		}
	})

	t.Run("returns master when main doesn't exist", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 3 && args[2] == "main" {
					return "", &mockError{}
				}
				if command == "git" && len(args) >= 3 && args[2] == "master" {
					return "abc123", nil
				}
				return "", nil
			},
		}

		result := getBaseBranch("/test", mock)
		if result != "master" {
			t.Errorf("expected master, got %s", result)
		}
	})

	t.Run("defaults to main when neither exists", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", &mockError{}
			},
		}

		result := getBaseBranch("/test", mock)
		if result != "main" {
			t.Errorf("expected main as default, got %s", result)
		}
	})
}

func TestGetDiffStats(t *testing.T) {
	t.Run("returns summary line", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return " file1.go | 10 +++++++---\n file2.go | 5 ++---\n 2 files changed, 10 insertions(+), 5 deletions(-)", nil
			},
		}

		result := getDiffStats("/test", "main", mock)
		if !strings.Contains(result, "2 files changed") {
			t.Errorf("expected diff summary, got %s", result)
		}
	})

	t.Run("returns empty for no changes", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		result := getDiffStats("/test", "main", mock)
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})
}

func TestGetFilesChanged(t *testing.T) {
	t.Run("returns file list", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "M\tfile1.go\nA\tfile2.go\nD\tfile3.go", nil
			},
		}

		result := getFilesChanged("/test", "main", mock)
		if !strings.Contains(result, "file1.go") {
			t.Error("expected file1.go in result")
		}
		if !strings.Contains(result, "file2.go") {
			t.Error("expected file2.go in result")
		}
	})

	t.Run("returns empty for no changes", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		result := getFilesChanged("/test", "main", mock)
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("with feature branch", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if dir != tmpDir {
					t.Errorf("expected dir %s, got %s", tmpDir, dir)
				}
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "feature/bd-123-test", nil
				}
				if command == "git" && len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--verify" {
					return "abc123", nil // main exists
				}
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "log" {
					return "abc123 Test commit\ndef456 Another commit", nil
				}
				if command == "git" && len(args) >= 1 && args[0] == "diff" {
					if len(args) >= 2 && args[1] == "--stat" {
						return " file.go | 10 +++++++---\n 1 file changed, 7 insertions(+), 3 deletions(-)", nil
					}
					if len(args) >= 2 && args[1] == "--name-status" {
						return "M\tfile.go", nil
					}
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

	t.Run("on main branch shows warning", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "main", nil
				}
				if command == "git" && len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--verify" {
					return "abc123", nil
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
		// Output will contain warning about being on main branch
	})

	t.Run("verbose mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "feature/test", nil
				}
				if command == "git" && len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--verify" {
					return "abc123", nil
				}
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

// mockError implements error interface for testing
type mockError struct{}

func (e *mockError) Error() string {
	return "mock error"
}

func TestGetExistingPR(t *testing.T) {
	t.Run("returns PR info when PR exists", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
					return `[{"number":42,"title":"Test PR","url":"https://github.com/test/repo/pull/42","state":"OPEN"}]`, nil
				}
				return "", nil
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result == nil {
			t.Fatal("expected PR info, got nil")
		}
		if result.Number != 42 {
			t.Errorf("expected PR number 42, got %d", result.Number)
		}
		if result.Title != "Test PR" {
			t.Errorf("expected title 'Test PR', got %s", result.Title)
		}
		if result.State != "OPEN" {
			t.Errorf("expected state 'OPEN', got %s", result.State)
		}
	})

	t.Run("returns nil when no PR exists", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "[]", nil
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})

	t.Run("returns nil on error", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", &mockError{}
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})

	t.Run("returns nil on invalid JSON", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "not valid json", nil
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestGetExistingPRProtocol(t *testing.T) {
	pr := &PRInfo{Number: 42, Title: "Test PR", URL: "https://github.com/test/repo/pull/42", State: "OPEN"}

	t.Run("non-verbose protocol", func(t *testing.T) {
		result := getExistingPRProtocol(pr, false)

		if !strings.Contains(result, "pull request already exists") {
			t.Error("expected existing PR message")
		}
		if !strings.Contains(result, "gh pr view 42") {
			t.Error("expected gh pr view command with PR number")
		}
		if !strings.Contains(result, "gh pr checks 42") {
			t.Error("expected gh pr checks command")
		}
		if !strings.Contains(result, "git push") {
			t.Error("expected git push instruction")
		}
	})

	t.Run("verbose protocol", func(t *testing.T) {
		result := getExistingPRProtocol(pr, true)

		if !strings.Contains(result, "**Review the PR status**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "gh pr view 42 --comments") {
			t.Error("expected comments command")
		}
		if !strings.Contains(result, "gh pr merge 42") {
			t.Error("expected merge command in verbose mode")
		}
		if !strings.Contains(result, "```bash") {
			t.Error("expected code blocks in verbose mode")
		}
	})
}
