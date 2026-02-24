package gitinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gi4nks/tecla/internal/deps"
	"github.com/gi4nks/tecla/internal/runner"
)

type Options struct {
	Timeout               time.Duration
	Workers               int
	StaleThresholdDays    int
	CustomRecommendations []CustomRecommendation
}

type CustomRecommendation struct {
	Condition string `json:"condition"`
	Text      string `json:"text"`
	Command   string `json:"command,omitempty"`
}

type ProgressFunc func()

type Recommendation struct {
	Text    string `json:"text"`
	Command string `json:"command,omitempty"`
}

type RepoInfo struct {
	Path            string           `json:"path"`
	Workspace       string           `json:"workspace"`
	IsRepo          bool             `json:"is_repo"`
	ModuleName      string           `json:"module_name"`
	Branch          string           `json:"branch"`
	Detached        bool             `json:"detached"`
	IsEmpty         bool             `json:"is_empty"`
	Status          StatusInfo       `json:"status"`
	Upstream        string           `json:"upstream"`
	Ahead           int              `json:"ahead"`
	Behind          int              `json:"behind"`
	Remote          string           `json:"remote"` // Primary remote URL (fallback)
	Remotes         []RemoteDetail   `json:"remotes"`
	StashCount      int              `json:"stash_count"`
	Submodules      SubmoduleInfo    `json:"submodules"`
	MergedBranches  []string         `json:"merged_branches"`
	RemoteStatus    RemoteStatus     `json:"remote_status"` // Primary remote status
	Dependencies    []string         `json:"dependencies"`
	Recommendations []Recommendation `json:"recommendations"`
	HealthScore     int              `json:"health_score"`
	LastCommitAt    time.Time        `json:"last_commit_at,omitempty"`
	Error           string           `json:"error,omitempty"`
	Errors          []error          `json:"-"` // Added for structured errors
}

func (info *RepoInfo) CalculateHealthScore() {
	score := 100
	if info.Error != "" {
		score -= 50
	}
	if info.Detached {
		score -= 20
	}
	if info.Ahead > 0 {
		score -= 10
	}
	if info.Behind > 0 {
		score -= 5
	}
	if !info.Status.Clean {
		score -= 10
	}
	if info.Status.Untracked {
		score -= 5
	}
	if info.StashCount > 0 {
		score -= 2
	}
	if info.Submodules.Dirty {
		score -= 5
	}
	if len(info.MergedBranches) > 0 {
		score -= minInt(20, len(info.MergedBranches)*2)
	}

	// Remote Health
	switch info.RemoteStatus.CIStatus {
	case "failure":
		score -= 30
	case "pending":
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	info.HealthScore = score
}

func (info *RepoInfo) DoctorRecommendations() []Recommendation {
	var recs []Recommendation
	for _, r := range info.Recommendations {
		if strings.HasPrefix(r.Text, "Cleanup") || strings.Contains(r.Text, "in progress") {
			recs = append(recs, r)
		}
	}
	return recs
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Custom error types for better diagnostics
type GitError struct {
	Op  string
	Err error
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s: %v", e.Op, e.Err)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

type TimeoutError struct {
	Op      string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("git %s: timed out after %v", e.Op, e.Timeout)
}

type StatusInfo struct {
	Clean     bool `json:"clean"`
	Modified  bool `json:"modified"`
	Untracked bool `json:"untracked"`
	Staged    bool `json:"staged"`
}

type SubmoduleInfo struct {
	Count int  `json:"count"`
	Dirty bool `json:"dirty"`
}

type RemoteStatus struct {
	CIStatus string `json:"ci_status"` // success, failure, pending, unknown, loading
	PRCount  int    `json:"pr_count"`
}

type RemoteDetail struct {
	Name   string       `json:"name"`
	URL    string       `json:"url"`
	Status RemoteStatus `json:"status"`
}

type GlobalConfig struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Version   string `json:"version"`
}

func GetGlobalConfig(ctx context.Context, timeout time.Duration) GlobalConfig {
	cfg := GlobalConfig{}

	out, err := runGit(ctx, ".", timeout, "version")
	if err == nil {
		cfg.Version = strings.TrimSpace(out)
	}

	out, err = runGit(ctx, ".", timeout, "config", "--get", "user.name")
	if err == nil {
		cfg.UserName = strings.TrimSpace(out)
	}

	out, err = runGit(ctx, ".", timeout, "config", "--get", "user.email")
	if err == nil {
		cfg.UserEmail = strings.TrimSpace(out)
	}

	return cfg
}

type operationState struct {
	Rebase     bool
	Merge      bool
	CherryPick bool
	Revert     bool
}

func Collect(ctx context.Context, repos []string, opts Options, progress ProgressFunc) []RepoInfo {
	if opts.Workers <= 0 {
		opts.Workers = 1
	}

	repoCh := make(chan string)
	resCh := make(chan RepoInfo)
	var wg sync.WaitGroup

	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoCh {
				resCh <- InspectRepo(ctx, repo, opts)
				if progress != nil {
					progress()
				}
			}
		}()
	}

	go func() {
		for _, repo := range repos {
			repoCh <- repo
		}
		close(repoCh)
		wg.Wait()
		close(resCh)
	}()

	var results []RepoInfo
	for res := range resCh {
		results = append(results, res)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results
}

func InspectRepo(ctx context.Context, repo string, opts Options) RepoInfo {
	modName, dependencies := deps.ScanRepo(repo)

	info := RepoInfo{
		Path:         repo,
		Workspace:    filepath.Base(filepath.Dir(repo)),
		IsRepo:       IsRepo(repo),
		ModuleName:   modName,
		Dependencies: dependencies,
	}

	if !info.IsRepo {
		info.CalculateHealthScore()
		info.Recommendations = buildNakedRecommendations(info)
		return info
	}

	// Collect all remotes
	remotes, _ := collectRemotes(ctx, repo, opts)
	info.Remotes = remotes
	if len(remotes) > 0 {
		// Set primary remote for backward compatibility
		info.Remote = remotes[0].URL
		for _, r := range remotes {
			if r.Name == "origin" {
				info.Remote = r.URL
				break
			}
		}
		info.RemoteStatus = RemoteStatus{CIStatus: "loading"}
	}
	
	statusOut, err := runGit(ctx, repo, opts.Timeout, "status", "--porcelain=v2", "-b")
	if err != nil {
		info.addError(&GitError{Op: "status", Err: err})
		return info
	}

	status, branch := parsePorcelainV2(statusOut)
	info.Status = status
	info.Branch = branch.Head
	info.Upstream = branch.Upstream
	info.Ahead = branch.Ahead
	info.Behind = branch.Behind
	info.IsEmpty = branch.IsInitial
	info.Detached = isDetachedBranch(info.Branch)

	if info.Branch == "" {
		head, headErr := runGit(ctx, repo, opts.Timeout, "rev-parse", "--abbrev-ref", "HEAD")
		if headErr == nil {
			info.Branch = strings.TrimSpace(head)
			info.Detached = isDetachedBranch(info.Branch)
		}
	}

	remote, remoteErr := remoteURL(ctx, repo, opts)
	if remoteErr != nil {
		info.addError(&GitError{Op: "remote", Err: remoteErr})
	}
	info.Remote = remote

	stashCount, stashErr := stashCount(ctx, repo, opts)
	if stashErr != nil {
		info.addError(&GitError{Op: "stash", Err: stashErr})
	}
	info.StashCount = stashCount

	submodules, subErr := submoduleInfo(ctx, repo, opts)
	if subErr != nil {
		info.addError(&GitError{Op: "submodule", Err: subErr})
	}
	info.Submodules = submodules

	merged, mergedErr := mergedBranches(ctx, repo, opts)
	if mergedErr == nil {
		info.MergedBranches = merged
	}

	ops, opsErr := detectOperations(ctx, repo, opts)
	if opsErr != nil {
		info.addError(&GitError{Op: "operations", Err: opsErr})
	}

	lastCommit, commitErr := runGit(ctx, repo, opts.Timeout, "log", "-1", "--format=%cI")
	if commitErr == nil {
		t, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(lastCommit))
		if parseErr == nil {
			info.LastCommitAt = t
		}
	}

	info.Recommendations = buildRecommendations(info, ops, opts, info.MergedBranches)
	info.CalculateHealthScore()
	return info
}

func (info *RepoInfo) addError(err error) {
	if err == nil {
		return
	}
	info.Errors = append(info.Errors, err)
	// Maintain backward compatibility with the Error string field
	if info.Error == "" {
		info.Error = err.Error()
	} else {
		info.Error = info.Error + "; " + err.Error()
	}
}

func isDetachedBranch(name string) bool {
	if name == "" {
		return false
	}
	if name == "HEAD" {
		return true
	}
	return strings.HasPrefix(name, "(detached")
}

func buildRecommendations(info RepoInfo, ops operationState, opts Options, merged []string) []Recommendation {
	var recs []Recommendation

	if len(merged) > 0 {
		recs = append(recs, Recommendation{
			Text:    fmt.Sprintf("Cleanup merged branches (%d): %s", len(merged), strings.Join(merged, ", ")),
			Command: "git branch -d " + strings.Join(merged, " "),
		})
	}

	if ops.Rebase {
		recs = append(recs, Recommendation{Text: "Rebase in progress", Command: "git rebase --continue"})
	}
	if ops.Merge {
		recs = append(recs, Recommendation{Text: "Merge in progress", Command: "git merge --continue"})
	}
	if ops.CherryPick {
		recs = append(recs, Recommendation{Text: "Cherry-pick in progress", Command: "git cherry-pick --continue"})
	}
	if ops.Revert {
		recs = append(recs, Recommendation{Text: "Revert in progress", Command: "git revert --continue"})
	}

	if info.IsEmpty {
		recs = append(recs, Recommendation{Text: "Create the first commit", Command: "git add -A && git commit -m \"initial commit\""})
	}
	if info.Detached {
		recs = append(recs, Recommendation{Text: "Create a branch", Command: "git switch -c <branch-name>"})
	}
	if isMainBranch(info.Branch) && !info.Detached && (info.Status.Modified || info.Status.Untracked || info.Status.Staged) {
		recs = append(recs, Recommendation{Text: "Work on a feature branch", Command: "git switch -c <branch-name>"})
	}
	if info.Status.Modified || info.Status.Untracked {
		recs = append(recs, Recommendation{Text: "Review changes and stage them", Command: "git add -A"})
	}
	if info.Status.Staged {
		recs = append(recs, Recommendation{Text: "Commit staged changes", Command: "git commit -m \"<message>\""})
	}
	if info.Upstream == "" && info.Branch != "" && !info.Detached && !info.IsEmpty {
		if info.Remote != "" {
			recs = append(recs, Recommendation{Text: "Set upstream by pushing", Command: fmt.Sprintf("git push -u origin %s", info.Branch)})
		} else {
			recs = append(recs, Recommendation{Text: "Set upstream", Command: "git branch --set-upstream-to <remote>/<branch>"})
		}
	}
	if info.Remote == "" {
		recs = append(recs, Recommendation{Text: "Add a remote", Command: "git remote add origin <url>"})
	}
	if info.Ahead > 0 && info.Behind > 0 && info.Upstream != "" && info.Branch != "" {
		recs = append(recs, Recommendation{Text: "Sync diverged branch", Command: fmt.Sprintf("git fetch origin && git rebase origin/%s", info.Branch)})
	}
	if info.Ahead > 0 {
		recs = append(recs, Recommendation{Text: "Push commits", Command: "git push"})
	}
	if info.Behind > 0 {
		if info.Status.Modified || info.Status.Untracked || info.Status.Staged {
			recs = append(recs, Recommendation{Text: "Stash before pulling", Command: "git stash push -u -m \"tecla\""})
		}
		recs = append(recs, Recommendation{Text: "Pull updates", Command: "git pull --rebase"})
	}
	if info.StashCount > 0 {
		if info.Status.Clean {
			recs = append(recs, Recommendation{Text: "Apply stash", Command: "git stash pop"})
		} else {
			recs = append(recs, Recommendation{Text: "Review stashes", Command: "git stash list"})
		}
	}
	if info.Submodules.Count > 0 && info.Submodules.Dirty {
		recs = append(recs, Recommendation{Text: "Update submodules", Command: "git submodule update --recursive"})
	}

	threshold := opts.StaleThresholdDays
	if threshold <= 0 {
		threshold = 30
	}
	if !info.LastCommitAt.IsZero() && time.Since(info.LastCommitAt) > time.Duration(threshold)*24*time.Hour {
		days := int(time.Since(info.LastCommitAt).Hours() / 24)
		recs = append(recs, Recommendation{Text: fmt.Sprintf("Stale repository: last commit was %d days ago", days)})
	}

	for _, cr := range opts.CustomRecommendations {
		if evaluateCondition(info, cr.Condition) {
			recs = append(recs, Recommendation{Text: cr.Text, Command: cr.Command})
		}
	}

	return recs
}

func evaluateCondition(info RepoInfo, condition string) bool {
	// Simple conditions for now based on strings
	switch condition {
	case "is_dirty":
		return !info.Status.Clean
	case "is_stale":
		// handled above or maybe here, assume is_dirty for now
		return !info.Status.Clean
	case "has_untracked":
		return info.Status.Untracked
	case "is_detached":
		return info.Detached
	}
	return false
}

func collectRemotes(ctx context.Context, repo string, opts Options) ([]RemoteDetail, error) {
	out, err := runGit(ctx, repo, opts.Timeout, "remote", "-v")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var remotes []RemoteDetail
	seen := make(map[string]bool)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[len(fields)-1] != "(fetch)" {
			continue
		}
		name := fields[0]
		url := fields[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, RemoteDetail{
			Name:   name,
			URL:    url,
			Status: RemoteStatus{CIStatus: "loading"},
		})
	}
	return remotes, nil
}

func remoteURL(ctx context.Context, repo string, opts Options) (string, error) {
	out, err := runGit(ctx, repo, opts.Timeout, "remote", "get-url", "origin")
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	if strings.Contains(err.Error(), "No such remote") {
		return "", nil
	}
	verbose, verboseErr := runGit(ctx, repo, opts.Timeout, "remote", "-v")
	if verboseErr != nil {
		return "", verboseErr
	}
	return parseRemoteVerbose(verbose), nil
}

func parseRemoteVerbose(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var fallback string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[len(fields)-1] == "(fetch)" {
			if fields[0] == "origin" {
				return fields[1]
			}
			if fallback == "" {
				fallback = fields[1]
			}
		}
	}
	return fallback
}

func stashCount(ctx context.Context, repo string, opts Options) (int, error) {
	out, err := runGit(ctx, repo, opts.Timeout, "stash", "list")
	if err != nil {
		return 0, err
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return 0, nil
	}
	return len(strings.Split(trimmed, "\n")), nil
}

func submoduleInfo(ctx context.Context, repo string, opts Options) (SubmoduleInfo, error) {
	out, err := runGit(ctx, repo, opts.Timeout, "submodule", "status", "--recursive")
	if err != nil {
		return SubmoduleInfo{}, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	info := SubmoduleInfo{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		info.Count++
		if line[0] != ' ' {
			info.Dirty = true
		}
	}
	return info, nil
}

func mergedBranches(ctx context.Context, repo string, opts Options) ([]string, error) {
	// git branch --merged returns branches merged into current HEAD
	out, err := runGit(ctx, repo, opts.Timeout, "branch", "--merged")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var merged []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "*") { // Skip current branch
			continue
		}
		if isMainBranch(line) { // Don't recommend deleting main/master
			continue
		}
		merged = append(merged, line)
	}
	return merged, nil
}

func FetchRemoteStatus(ctx context.Context, repo string, remoteURL string, timeout time.Duration) (RemoteStatus, error) {
	status := RemoteStatus{CIStatus: "unknown"}

	remoteTimeout := timeout
	if remoteTimeout < 10*time.Second {
		remoteTimeout = 10 * time.Second
	}

	if !strings.Contains(remoteURL, "github.com") {
		return status, nil
	}

	// Fetch PR Count
	prOut, prErr := runner.GlobalRunner.Run(ctx, repo, remoteTimeout, "gh", "pr", "list", "--json", "number")
	if prErr != nil {
		return status, fmt.Errorf("gh pr list: %w", prErr)
	}
	
	if prOut != "" {
		var prs []interface{}
		if err := json.Unmarshal([]byte(prOut), &prs); err == nil {
			status.PRCount = len(prs)
		}
	}

	// Fetch CI Status
	ciOut, ciErr := runner.GlobalRunner.Run(ctx, repo, remoteTimeout, "gh", "run", "list", "--limit", "1", "--json", "conclusion")
	if ciErr != nil {
		return status, fmt.Errorf("gh run list: %w", ciErr)
	}

	if ciOut != "" {
		var runs []struct {
			Conclusion string `json:"conclusion"`
		}
		if err := json.Unmarshal([]byte(ciOut), &runs); err == nil && len(runs) > 0 {
			conclusion := strings.ToLower(runs[0].Conclusion)
			switch conclusion {
			case "success":
				status.CIStatus = "success"
			case "failure", "timed_out", "cancelled":
				status.CIStatus = "failure"
			case "pending", "in_progress", "queued", "":
				status.CIStatus = "pending"
			default:
				status.CIStatus = "unknown"
			}
		}
	}

	return status, nil
}

func runGit(ctx context.Context, repo string, timeout time.Duration, args ...string) (string, error) {
	cmdArgs := append([]string{"git"}, args...)
	return runner.GlobalRunner.Run(ctx, repo, timeout, cmdArgs...)
}

func isMainBranch(name string) bool {
	switch name {
	case "main", "master":
		return true
	default:
		return false
	}
}

func detectOperations(ctx context.Context, repo string, opts Options) (operationState, error) {
	state := operationState{}
	gitDir, err := gitDirPath(ctx, repo, opts)
	if err != nil {
		return state, err
	}

	if exists(filepath.Join(gitDir, "rebase-apply")) || exists(filepath.Join(gitDir, "rebase-merge")) {
		state.Rebase = true
	}
	if exists(filepath.Join(gitDir, "MERGE_HEAD")) {
		state.Merge = true
	}
	if exists(filepath.Join(gitDir, "CHERRY_PICK_HEAD")) {
		state.CherryPick = true
	}
	if exists(filepath.Join(gitDir, "REVERT_HEAD")) {
		state.Revert = true
	}

	return state, nil
}

func gitDirPath(ctx context.Context, repo string, opts Options) (string, error) {
	out, err := runGit(ctx, repo, opts.Timeout, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(out)
	if path == "" {
		return "", fmt.Errorf("empty git dir path")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(repo, path), nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return true
	}
	if info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return false
}

func buildNakedRecommendations(info RepoInfo) []Recommendation {
	name := filepath.Base(info.Path)
	return []Recommendation{
		{Text: "Initialize repository", Command: "git init"},
		{Text: "First commit", Command: "git add -A && git commit -m \"initial commit\""},
		{
			Text:    "Publish to GitHub",
			Command: fmt.Sprintf("gh repo create gi4nks/%s --private --source=. --remote=origin --push", name),
		},
	}
}
