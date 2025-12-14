package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitEnv creates a temp directory and initializes a GitService.
// It also configures local git identity to ensure commits work in CI environments.
func setupGitEnv(t *testing.T) (string, *GitService) {
	t.Helper()

	// 1. Create temporary directory
	tmpDir := t.TempDir()

	// 2. Initialize Service
	svc := NewGitService(tmpDir)

	// 3. Initialize Git Repo
	if err := svc.Init(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// 4. Configure local identity (required for commits to succeed)
	configureGitIdentity(t, tmpDir)

	return tmpDir, svc
}

func configureGitIdentity(t *testing.T, dir string) {
	runCmd(t, dir, "git", "config", "user.email", "test@lx-cli.com")
	runCmd(t, dir, "git", "config", "user.name", "LX Test Bot")
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Command failed: %s %v\nOutput: %s\nError: %v", name, args, out, err)
	}
}

func TestGitService_Init(t *testing.T) {
	// We manually create a dir to test Init specifically
	tmpDir := t.TempDir()
	svc := NewGitService(tmpDir)

	// 1. Run Init
	err := svc.Init()
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	// 2. Verify .git directory exists
	gitDir := filepath.Join(tmpDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf("Expected .git directory to be created, but it was missing")
	}

	// 3. Run Init again (idempotency check)
	err = svc.Init()
	if err != nil {
		t.Errorf("Subsequent Init() call should not fail: %v", err)
	}
}

func TestGitService_CommitChanges(t *testing.T) {
	dir, svc := setupGitEnv(t)

	// 1. Create a dummy file
	testFile := filepath.Join(dir, "note.md")
	err := os.WriteFile(testFile, []byte("# Test Note"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 2. Commit Changes
	commitMsg := "Add test note"
	err = svc.CommitChanges(commitMsg)
	if err != nil {
		t.Fatalf("CommitChanges() failed: %v", err)
	}

	// 3. Verify Commit exists via git log
	cmd := exec.Command("git", "log", "--oneline", "-n", "1")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read git log: %v", err)
	}

	if !strings.Contains(string(output), commitMsg) {
		t.Errorf("Expected log to contain '%s', got: %s", commitMsg, string(output))
	}

	// 4. Verify Status is clean
	status, err := svc.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if strings.TrimSpace(status) != "" {
		t.Errorf("Expected clean status after commit, got:\n%s", status)
	}
}

func TestGitService_CommitChanges_NoChanges(t *testing.T) {
	_, svc := setupGitEnv(t)

	// 1. Commit without changing anything
	// Depending on your implementation, this might error or succeed.
	// The implementation provided earlier returns an error if git commit fails.
	// However, usually "nothing to commit" exits with code 1.

	// If you want it to fail gracefully:
	err := svc.CommitChanges("Should fail or be ignored")

	// Note: Standard git behavior exits 1 if nothing to commit.
	// If your service returns that error, this test expects an error.
	if err == nil {
		// Strictly speaking, if we allow empty commits this passes.
		// If we don't, this might be expected to fail.
		// For now, let's just log it.
		t.Log("Commit with no changes did not return error (did you use --allow-empty?)")
	}
}

func TestGitService_Sync(t *testing.T) {
	// This requires simulating a Remote. We do this by creating a "bare" repo locally.

	// 1. Create "Remote" (Bare Repo)
	remoteDir := t.TempDir()
	runCmd(t, remoteDir, "git", "init", "--bare")

	// 2. Create "Local" Repo
	localDir := t.TempDir()

	// Clone the bare repo to local
	runCmd(t, localDir, "git", "clone", remoteDir, ".")

	// Configure Identity
	configureGitIdentity(t, localDir)

	// 3. Init Service on Local
	svc := NewGitService(localDir)

	// 4. Create a commit locally
	testFile := filepath.Join(localDir, "sync_test.md")
	_ = os.WriteFile(testFile, []byte("Sync Content"), 0644)

	if err := svc.CommitChanges("Pre-sync commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// 5. Perform Sync (Push)
	if err := svc.Sync(); err != nil {
		t.Fatalf("Sync() failed: %v", err)
	}

	// 6. Verify "Remote" received the commit
	// We can check by cloning "Remote" to a 3rd dir or checking log in bare repo
	logCmd := exec.Command("git", "log", "--oneline")
	logCmd.Dir = remoteDir
	out, err := logCmd.Output()
	if err != nil {
		t.Fatalf("Failed to read remote log: %v", err)
	}

	if !strings.Contains(string(out), "Pre-sync commit") {
		t.Errorf("Remote repo did not receive the commit via Sync")
	}
}
