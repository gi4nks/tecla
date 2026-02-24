package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gi4nks/tecla/gitinfo"
)

type Options struct {
	Roots              []string
	IncludeHidden      bool
	ExcludePatterns    []string
	DefaultIgnoredDirs []string
	MaxDepth           int
}

func Scan(opts Options) ([]string, []error) {
	repos, _, errs := ScanAll(opts)
	return repos, errs
}

func ScanAll(opts Options) ([]string, []string, []error) {
	roots := opts.Roots
	if len(roots) == 0 {
		roots = []string{"."}
	}

	var repos []string
	var dirs []string
	var errs []error

	for _, root := range roots {
		r, d, e := ScanOne(root, opts)
		repos = append(repos, r...)
		dirs = append(dirs, d...)
		errs = append(errs, e...)
	}

	return repos, dirs, errs
}

func ScanOne(root string, opts Options) ([]string, []string, []error) {
	absRoot, err := filepath.Abs(root)
	if err == nil {
		root = absRoot
	}

	var repos []string
	var dirs []string
	var errs []error

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, walkErr))
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		base := d.Name()
		if base == ".git" {
			return filepath.SkipDir
		}

		depth := pathDepth(root, path)
		if opts.MaxDepth >= 0 && depth > opts.MaxDepth {
			return filepath.SkipDir
		}

		if shouldSkipDir(base, path, root, opts) {
			return filepath.SkipDir
		}

		if path != root {
			dirs = append(dirs, path)
		}

		if gitinfo.IsRepo(path) {
			repos = append(repos, path)
			return filepath.SkipDir
		}

		if path != root && pathDepth(root, path) == 1 {
			repos = append(repos, path)
		}

		if opts.MaxDepth >= 0 && depth == opts.MaxDepth && path != root {
			return filepath.SkipDir
		}

		return nil
	})

	return repos, dirs, errs
}

func shouldSkipDir(base, path, root string, opts Options) bool {
	if path != root {
		for _, ignored := range opts.DefaultIgnoredDirs {
			if base == ignored {
				return true
			}
		}

		if !opts.IncludeHidden && strings.HasPrefix(base, ".") {
			return true
		}
	}

	if len(opts.ExcludePatterns) == 0 {
		return false
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	base = filepath.ToSlash(base)

	for _, pattern := range opts.ExcludePatterns {
		if pattern == "" {
			continue
		}

		// If pattern is absolute, check if path matches it or is inside it
		if filepath.IsAbs(pattern) {
			if path == pattern || strings.HasPrefix(path, pattern+string(filepath.Separator)) {
				return true
			}
			continue
		}

		pattern = filepath.ToSlash(pattern)
		if matchPattern(pattern, base) || matchPattern(pattern, rel) {
			return true
		}
	}

	return false
}

func matchPattern(pattern, value string) bool {
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	return matched
}

func pathDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	rel = filepath.ToSlash(rel)
	return strings.Count(rel, "/") + 1
}
