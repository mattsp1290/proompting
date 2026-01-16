package git

import (
	"errors"
	"strings"
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

		result := GetCurrentBranch("/test/dir", mock)
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

		result := GetCurrentBranch("/test/dir", mock)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestGetStatusCounts(t *testing.T) {
	testCases := []struct {
		name     string
		status   string
		expected StatusCounts
	}{
		{"clean", "", StatusCounts{}},
		{"staged only", "A  file.go", StatusCounts{Staged: 1}},
		{"modified only", "MM file.go", StatusCounts{Staged: 1, Modified: 1}},
		{"untracked only", "?? file.go", StatusCounts{Untracked: 1}},
		{"mixed", "A  a.go\n M b.go\n?? c.go", StatusCounts{Staged: 1, Modified: 1, Untracked: 1}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockRunner{
				RunFunc: func(dir string, command string, args ...string) (string, error) {
					return tc.status, nil
				},
			}

			result := GetStatusCounts("/test/dir", mock)
			if result != tc.expected {
				t.Errorf("GetStatusCounts() = %+v, want %+v", result, tc.expected)
			}
		})
	}
}

func TestFormatStatusCounts(t *testing.T) {
	testCases := []struct {
		name     string
		counts   StatusCounts
		expected string
	}{
		{"empty", StatusCounts{}, ""},
		{"staged only", StatusCounts{Staged: 1}, "1 staged"},
		{"modified only", StatusCounts{Modified: 2}, "2 modified"},
		{"untracked only", StatusCounts{Untracked: 3}, "3 untracked"},
		{"all", StatusCounts{Staged: 1, Modified: 2, Untracked: 3}, "1 staged, 2 modified, 3 untracked"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatStatusCounts(tc.counts)
			if result != tc.expected {
				t.Errorf("FormatStatusCounts(%+v) = %q, want %q", tc.counts, result, tc.expected)
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

		result := GetBranchCommits("/test/dir", "feature/test", mock)
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

		result := GetBranchCommits("/test/dir", "main", mock)
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

		result := GetBranchCommits("/test/dir", "feature/test", mock)
		if !strings.Contains(result, "abc123 Commit from master") {
			t.Errorf("expected master fallback, got %q", result)
		}
	})
}

func TestGetStashCount(t *testing.T) {
	t.Run("no stashes", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		result := GetStashCount("/test/dir", mock)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("multiple stashes", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "stash@{0}: WIP\nstash@{1}: WIP", nil
			},
		}

		result := GetStashCount("/test/dir", mock)
		if result != 2 {
			t.Errorf("expected 2, got %d", result)
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

		result := CheckRemoteStatus("/test/dir", mock, false)
		if result.Behind != 3 {
			t.Errorf("expected behind=3, got %d", result.Behind)
		}
		if !strings.Contains(result.Info, "behind") {
			t.Errorf("expected info to contain 'behind', got %q", result.Info)
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

		result := CheckRemoteStatus("/test/dir", mock, false)
		if result.Ahead != 2 {
			t.Errorf("expected ahead=2, got %d", result.Ahead)
		}
	})

	t.Run("detects ahead and behind", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 1 && args[0] == "status" {
					return "## feature/test...origin/feature/test [ahead 1, behind 2]", nil
				}
				return "", nil
			},
		}

		result := CheckRemoteStatus("/test/dir", mock, false)
		if result.Ahead != 1 || result.Behind != 2 {
			t.Errorf("expected ahead=1, behind=2, got ahead=%d, behind=%d", result.Ahead, result.Behind)
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
			result := CountLines(tc.input)
			if result != tc.expected {
				t.Errorf("CountLines(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}
