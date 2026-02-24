package gitinfo_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gi4nks/tecla/gitinfo"
)

func runGitCmd(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\noutput: %s", args, err, out)
	}
}

func setupTempRepo(t *testing.T) string {
	dir := t.TempDir()
	
	// Create a stable git environment for tests
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.name", "Test User")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "commit.gpgsign", "false")
	
	return dir
}

func TestInspectRepo_Clean(t *testing.T) {
	dir := setupTempRepo(t)
	
	// Make a first commit
	err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "init")

	info := gitinfo.InspectRepo(context.Background(), dir, gitinfo.Options{Timeout: 5 * time.Second})

	if info.Error != "" {
		t.Fatalf("unexpected error: %s", info.Error)
	}
	if !info.Status.Clean {
		t.Fatalf("expected clean repo")
	}
	if info.IsEmpty {
		t.Fatalf("expected non-empty repo")
	}
}

func TestInspectRepo_Dirty(t *testing.T) {
	dir := setupTempRepo(t)
	
	err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "init")

	err = os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	info := gitinfo.InspectRepo(context.Background(), dir, gitinfo.Options{Timeout: 5 * time.Second})

	if info.Error != "" {
		t.Fatalf("unexpected error: %s", info.Error)
	}
	if info.Status.Clean {
		t.Fatalf("expected dirty repo")
	}
	if !info.Status.Modified {
		t.Fatalf("expected modified status")
	}
}

func TestInspectRepo_Detached(t *testing.T) {
	dir := setupTempRepo(t)
	
	err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "init")
	
	err = os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello 2"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "second")

	runGitCmd(t, dir, "checkout", "HEAD~1")

	info := gitinfo.InspectRepo(context.Background(), dir, gitinfo.Options{Timeout: 5 * time.Second})

	if !info.Detached {
		t.Fatalf("expected detached repo")
	}
}
