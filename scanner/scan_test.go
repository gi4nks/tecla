package scanner

import "testing"

func TestShouldSkipDirDefaults(t *testing.T) {
	opts := Options{DefaultIgnoredDirs: []string{"node_modules"}}
	if !shouldSkipDir("node_modules", "/root/node_modules", "/root", opts) {
		t.Fatalf("expected node_modules to be skipped")
	}
}

func TestShouldSkipDirHidden(t *testing.T) {
	opts := Options{IncludeHidden: false}
	if !shouldSkipDir(".hidden", "/root/.hidden", "/root", opts) {
		t.Fatalf("expected hidden dir to be skipped")
	}

	opts.IncludeHidden = true
	if shouldSkipDir(".hidden", "/root/.hidden", "/root", opts) {
		t.Fatalf("expected hidden dir to be included")
	}
}

func TestShouldSkipDirExcludePatterns(t *testing.T) {
	opts := Options{ExcludePatterns: []string{"vendor", "logs/*"}}
	if !shouldSkipDir("vendor", "/root/vendor", "/root", opts) {
		t.Fatalf("expected vendor to be excluded")
	}
	if !shouldSkipDir("app", "/root/logs/app", "/root", opts) {
		t.Fatalf("expected logs/app to be excluded")
	}
	if shouldSkipDir("src", "/root/src", "/root", opts) {
		t.Fatalf("did not expect src to be excluded")
	}
}
