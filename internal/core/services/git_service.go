package services

import (
	"fmt"
	"os"
	"os/exec"
)

// GitService handles interactions with the git CLI
type GitService struct {
	workingDir string
}

// NewGitService creates a new instance of GitService
func NewGitService(workingDir string) *GitService {
	return &GitService{
		workingDir: workingDir,
	}
}

// runGit executes a git command in the service's working directory
func (s *GitService) runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.workingDir
	// We silence Stdout to keep the CLI clean, but capture Stderr for errors
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Init initializes a new git repository if one doesn't exist
func (s *GitService) Init() error {
	// Check if .git directory already exists
	if _, err := os.Stat(s.workingDir + "/.git"); err == nil {
		return nil // Already initialized
	}
	return s.runGit("init")
}

// CommitChanges stages all files and commits them with the given message
func (s *GitService) CommitChanges(message string) error {
	// 1. Add all changes (including new files)
	if err := s.runGit("add", "."); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// 2. Check if there are actually changes to commit
	// "git diff-index --quiet HEAD" returns exit code 1 if there are changes.
	// We ignore the error here because we WANT it to fail (meaning changes exist).
	// However, if it succeeds (exit 0), it means no changes, so we can return early.
	// NOTE: If HEAD doesn't exist (fresh repo), this check might fail awkwardly,
	// so we'll skip the check for v1 and just try to commit.

	// 3. Commit
	// We use --allow-empty to prevent errors if the state didn't actually change,
	// though usually we want to know. For auto-backup, it's safer to just try.
	if err := s.runGit("commit", "-m", message); err != nil {
		// If the error is just "nothing to commit", we might want to suppress it,
		// but for now, returning the error is safer for debugging.
		return fmt.Errorf("git commit failed: %w", err)
	}

	return nil
}

// Sync pulls remote changes and pushes local commits
func (s *GitService) Sync() error {
	// 1. Pull with rebase to keep history clean
	// We ignore errors here because the remote might not exist or be empty
	_ = s.runGit("pull", "--rebase")

	// 2. Push changes
	if err := s.runGit("push"); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	return nil
}

// Status returns the current status of the repo (simplified)
func (s *GitService) Status() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = s.workingDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
