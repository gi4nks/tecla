package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRepo_Go(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tecla-test-*")
	defer os.RemoveAll(tmpDir)

	goModContent := `module github.com/user/testrepo

go 1.21

require (
	github.com/user/lib1 v1.0.0
	github.com/user/lib2 v0.5.0
)
`
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)

	modName, deps := ScanRepo(tmpDir)

	if modName != "github.com/user/testrepo" {
		t.Errorf("expected module name github.com/user/testrepo, got %s", modName)
	}

	expectedDeps := []string{"github.com/user/lib1", "github.com/user/lib2"}
	if len(deps) != len(expectedDeps) {
		t.Errorf("expected %d dependencies, got %d: %v", len(expectedDeps), len(deps), deps)
	}
}

func TestScanRepo_Node(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tecla-test-*")
	defer os.RemoveAll(tmpDir)

	packageJsonContent := `{
  "name": "my-cool-app",
  "dependencies": {
    "express": "^4.18.2",
    "lodash": "^4.17.21"
  }
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJsonContent), 0644)

	modName, deps := ScanRepo(tmpDir)

	if modName != "my-cool-app" {
		t.Errorf("expected module name my-cool-app, got %s", modName)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestUnique(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b"}
	output := unique(input)
	if len(output) != 3 {
		t.Errorf("expected 3 unique elements, got %d", len(output))
	}
}
