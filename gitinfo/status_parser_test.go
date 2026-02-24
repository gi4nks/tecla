package gitinfo

import "testing"

func TestParsePorcelainV2(t *testing.T) {
	output := `# branch.oid 1234567
# branch.head main
# branch.upstream origin/main
# branch.ab +2 -1
1 .M N... 100644 100644 100644 abcdef1 abcdef2 file1
1 M. N... 100644 100644 100644 abcdef1 abcdef2 file2
? newfile.txt
`

	status, branch := parsePorcelainV2(output)
	if branch.Head != "main" {
		t.Fatalf("expected branch head main, got %q", branch.Head)
	}
	if branch.Upstream != "origin/main" {
		t.Fatalf("expected upstream origin/main, got %q", branch.Upstream)
	}
	if branch.Ahead != 2 || branch.Behind != 1 {
		t.Fatalf("expected ahead/behind 2/1, got %d/%d", branch.Ahead, branch.Behind)
	}
	if !status.Modified || !status.Staged || !status.Untracked {
		t.Fatalf("expected modified/staged/untracked true, got %+v", status)
	}
	if status.Clean {
		t.Fatalf("expected clean false")
	}
}

func TestParsePorcelainV2Initial(t *testing.T) {
	output := `# branch.oid (initial)
# branch.head main
`

	_, branch := parsePorcelainV2(output)
	if !branch.IsInitial {
		t.Fatalf("expected initial branch detected")
	}
}
